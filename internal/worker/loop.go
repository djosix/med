package worker

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
	"github.com/djosix/med/internal/readwriter"
	"google.golang.org/protobuf/proto"
)

var (
	loopLogger = logger.NewLogger("Loop")

	ErrLoopClosed = fmt.Errorf("loop closed")
)

type Loop interface {
	Run()                                                                 // Run the loop
	Start(h Proc) (procID uint32, doneCh <-chan struct{})                 // Start a Proc
	StartLater(p Proc) (procID uint32, handle func(bool) <-chan struct{}) // Start a Proc later
	Remove(id uint32) bool                                                // Remove a Proc
	Done() <-chan struct{}                                                // Get the done chan of loop ctx
	Stop()                                                                // Stop the loop
}

type LoopImpl struct {
	lastProcID uint32
	procData   map[uint32]loopProcInfo
	procLock   sync.Mutex
	procCtx    context.Context
	procCancel context.CancelFunc
	procWg     sync.WaitGroup

	runLock sync.Mutex

	pktInCh         chan *pb.Packet
	pktOutCh        chan *pb.Packet
	frameRw         readwriter.FrameReadWriter
	dispatchTimeout time.Duration

	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once
}

type loopProcInfo struct {
	proc    Proc
	ctx     context.Context
	cancel  context.CancelFunc
	pktInCh chan *pb.Packet
}

func NewLoop(ctx context.Context, rw io.ReadWriter) *LoopImpl {
	var frameRw readwriter.FrameReadWriter
	frameRw = readwriter.NewPlainFrameReadWriter(rw)
	frameRw = readwriter.NewSnappyFrameReadWriter(frameRw) // compress frames

	var cancel context.CancelFunc
	loopCtx, loopCancel := context.WithCancel(ctx)
	procCtx, procCancel := context.WithCancel(ctx)

	return &LoopImpl{
		lastProcID: 0,
		procData:   map[uint32]loopProcInfo{},
		procLock:   sync.Mutex{},
		procCtx:    procCtx,
		procCancel: procCancel,

		runLock: sync.Mutex{},

		pktInCh:         make(chan *pb.Packet, 0),
		pktOutCh:        make(chan *pb.Packet, 0),
		frameRw:         frameRw,
		dispatchTimeout: time.Duration(1) * time.Second,

		ctx:    loopCtx,
		cancel: loopCancel,
	}
}

func (loop *LoopImpl) Run() {
	loop.runLock.Lock()
	defer loop.runLock.Unlock()

	logger := loopLogger.NewLogger("Run")
	logger.Debug("start")
	defer logger.Debug("done")

	for _, loopFunc := range []func(){
		loop.reader,
		loop.dispatcher,
		loop.writer,
	} {
		loopFunc := loopFunc
		loop.wg.Add(1)
		go func() {
			defer loop.wg.Done()
			defer loop.Stop() // stop all other loops after any ends
			loopFunc()
		}()
	}
	loop.wg.Wait()

	if conn, ok := loop.ctx.Value("conn").(net.Conn); ok {
		logger.Debug("RefreshIO")
		helper.RefreshIO(conn)
	} else {
		panic("cannot find conn in ctx")
	}
}

func (loop *LoopImpl) Start(p Proc) (procID uint32, doneCh <-chan struct{}) {
	procID, handle := loop.StartLater(p)
	return procID, handle(true)
}

func (loop *LoopImpl) StartLater(p Proc) (procID uint32, handle func(bool) <-chan struct{}) {
	if p == nil {
		panic("p is nil")
	}

	loop.procLock.Lock()
	defer loop.procLock.Unlock()

	// Get a valid procID
	{
		procID = loop.lastProcID + 1
		for {
			_, exists := loop.procData[procID]
			if !exists && procID != 0 { // procID 0 is reserved
				break
			}
			procID++
		}
		loop.lastProcID = procID
	}

	ctx, cancel := context.WithCancel(loop.ctx)
	pktInCh := make(chan *pb.Packet)
	pktOutCh := make(chan *pb.Packet)

	loop.procData[procID] = loopProcInfo{
		proc:    p,
		ctx:     ctx,
		cancel:  cancel,
		pktInCh: pktInCh,
	}

	sender := func() {
		logger := loopLogger.NewLogger(fmt.Sprintf("sender[proc=%v]", procID))
		logger.Debug("start")
		defer logger.Debug("done")

		for {
			pkt := <-pktOutCh
			if pkt == nil {
				logger.Debug("packet output channel closed")
				return
			}

			// logger.Debugf("packet: [%v]", pkt)

			// Must be the same.
			pkt.SourceID = procID
			pkt.TargetID = procID

			select {
			case loop.pktOutCh <- pkt:
			case <-loop.ctx.Done():
				return
			}
		}
	}

	start := func() {
		logger := loopLogger.NewLogger(fmt.Sprintf("proc[%v]", procID))
		logger.Debugf("start kind=%v", p.Kind())
		defer logger.Debug("done")
		defer loop.Remove(procID)
		defer close(pktOutCh) // must close after proc ends

		loop.wg.Add(1)
		go func() {
			defer loop.wg.Done()
			sender()
		}()

		runCtx := ProcRunCtx{
			Context:  ctx,
			Cancel:   cancel,
			Loop:     loop,
			PktInCh:  pktInCh,
			PktOutCh: pktOutCh,
			ProcID:   procID,
		}
		p.Run(&runCtx)
	}

	handle = func(ok bool) <-chan struct{} {
		if !ok {
			logger.Debugf("cancel handle for proc[%v]", procID)
			loop.Remove(procID)
			return nil
		}

		doneCh := make(chan struct{}, 0)

		loop.wg.Add(1)
		go func() {
			defer loop.wg.Done()
			defer close(doneCh)
			start()
		}()

		return doneCh
	}

	return procID, handle
}

func (loop *LoopImpl) Remove(procID uint32) bool {
	loop.procLock.Lock()
	defer loop.procLock.Unlock()

	loopLogger.Debugf("remove proc[%v]", procID)

	pd, ok := loop.procData[procID]
	if ok {
		loopLogger.Debugf("cleanup proc[%v]", procID)

		pd.cancel()
		close(pd.pktInCh)
		delete(loop.procData, procID)

		// Close remote proc
		go func() {
			select {
			case loop.pktOutCh <- pb.NewCtrlPacket(procID, pb.PacketCtrl_Exit):
			case <-loop.ctx.Done():
				return
			}
		}()
	}

	// Shutdown loop if no proc exists
	if len(loop.procData) == 0 {
		loopLogger.Debugf("shutdown loop")
		go loop.Stop() // avoid deadlock
	}

	return ok
}

func (loop *LoopImpl) Done() <-chan struct{} {
	return loop.ctx.Done()
}

func (loop *LoopImpl) Stop() {
	loop.stopOnce.Do(func() {
		loop.procLock.Lock()
		defer loop.procLock.Unlock()

		loop.cancel()

		for procID := range loop.procData {
			go loop.Remove(procID)
		}

		if conn, ok := loop.ctx.Value("conn").(net.Conn); ok {
			loopLogger.Debug("BreakIO")
			helper.BreakIO(conn)
		} else {
			panic("cannot find conn in ctx")
		}
	})
}

func (loop *LoopImpl) reader() {
	logger := loopLogger.NewLogger("reader")
	logger.Debug("start")
	defer logger.Debug("done")
	defer close(loop.pktInCh)

	for {
		frame, err := loop.frameRw.ReadFrame()
		if err != nil {
			logger.Debug("read frame:", err)
			return
		}
		pkt := pb.Packet{}
		if err = proto.Unmarshal(frame, &pkt); err != nil {
			logger.Warn("Unmarshal(frame, pb.Packet):", err)
			continue
		}

		select {
		case loop.pktInCh <- &pkt:
		case <-loop.Done():
			return
		}
	}
}

// dispatcher dispatches the incoming MedPkt.Message to their target processors with timeout
func (loop *LoopImpl) dispatcher() {
	logger := loopLogger.NewLogger("dispatcher")
	logger.Debug("start")
	defer logger.Debug("done")

	dispatchToProc := func(pkt *pb.Packet) error {
		loop.procLock.Lock()
		defer loop.procLock.Unlock()

		// logger.Debugf("packet: [%v]", pkt)
		pd, ok := loop.procData[pkt.TargetID]
		if !ok {
			return fmt.Errorf("proc[%v] not found", pkt.TargetID)
		}

		select {
		case pd.pktInCh <- pkt:
		case <-pd.ctx.Done():
			return nil
		case <-loop.Done():
			return ErrLoopClosed
		case <-time.NewTimer(loop.dispatchTimeout).C:
			return fmt.Errorf("dispatch timeout")
		}
		return nil
	}

	for pkt := range loop.pktInCh {
		// Handle exit packet from remote loop
		if pkt.SourceID == 0 && pb.IsPacketWithCtrlKind(pkt, pb.PacketCtrl_Exit) {
			logger.Debug("exit packet:", pkt)
			if pkt.TargetID == 0 {
				loop.Stop()
				continue
			}
			loop.Remove(pkt.TargetID)
			continue
		}

		// Handle error packet
		if pkt.Kind == pb.PacketKind_PacketKindError {
			logger.Errorf("remote proc[%v] error: %v", pkt.SourceID, string(pkt.Data))
			logger.Debug("remote error packet [%v]", pkt.String())
			if pkt.SourceID == 0 || pkt.SourceID == pkt.TargetID {
				loop.Remove(pkt.TargetID)
			}
			continue
		}

		err := dispatchToProc(pkt)
		if err != nil {
			logger.Debugf("dispatch error for [%v]: %v", pkt, err)

			errPkt := &pb.Packet{
				TargetID: pkt.SourceID,
				SourceID: pkt.TargetID,
				Kind:     pb.PacketKind_PacketKindError,
				Data:     []byte(err.Error()),
			}
			select {
			case loop.pktOutCh <- errPkt:
			case <-loop.ctx.Done():
				return
			}
		}
	}
}

func (loop *LoopImpl) writer() {
	logger := loopLogger.NewLogger("writer")
	logger.Debug("start")
	defer logger.Debug("done")

	for {
		select {
		case pkt, ok := <-loop.pktOutCh:
			if !ok {
				logger.Error("packet output closed")
				return
			}

			buf, err := proto.Marshal(pkt)
			if err != nil {
				panic("cannot encode packet")
			}

			err = loop.frameRw.WriteFrame(buf)
			if err != nil {
				logger.Error("write frame:", err)
				return
			}
		case <-loop.ctx.Done():
			return
		}
	}
}
