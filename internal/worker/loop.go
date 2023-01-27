package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

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
	pktOutCh        chan *pb.Packet // will have multiple senders so never close it
	frameRw         readwriter.FrameReadWriter
	closer          io.Closer
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

func NewLoop(ctx context.Context, rwc io.ReadWriteCloser) *LoopImpl {
	var frameRw readwriter.FrameReadWriter
	frameRw = readwriter.NewPlainFrameReadWriter(rwc)
	frameRw = readwriter.NewSnappyFrameReadWriter(frameRw) // compress frames

	loopCtx, loopCancel := context.WithCancel(ctx)
	procCtx, procCancel := context.WithCancel(ctx)

	return &LoopImpl{
		lastProcID: 0,
		procData:   map[uint32]loopProcInfo{},
		procLock:   sync.Mutex{},
		procCtx:    procCtx,
		procCancel: procCancel,
		procWg:     sync.WaitGroup{},

		runLock: sync.Mutex{},

		pktInCh:         make(chan *pb.Packet, 0),
		pktOutCh:        make(chan *pb.Packet, 0),
		frameRw:         frameRw,
		closer:          rwc,
		dispatchTimeout: time.Duration(1) * time.Second,

		wg:       sync.WaitGroup{},
		ctx:      loopCtx,
		cancel:   loopCancel,
		stopOnce: sync.Once{},
	}
}

func (loop *LoopImpl) Run() {
	loop.runLock.Lock()
	defer loop.runLock.Unlock()

	logger := loopLogger.NewLogger("Run")
	logger.Debug("start")
	defer logger.Debug("done")

	for _, loopFunc := range []func(){loop.reader, loop.dispatcher, loop.writer} {
		loopFunc := loopFunc
		loop.wg.Add(1)
		go func() {
			defer loop.wg.Done()
			defer loop.Stop() // stop all other loops after any ends
			loopFunc()
		}()
	}
	loop.wg.Wait()

	// if conn, ok := loop.ctx.Value("conn").(net.Conn); ok {
	// 	logger.Debug("RefreshIO")
	// 	helper.RefreshIO(conn)
	// } else {
	// 	panic("cannot find conn in ctx")
	// }
}

func (loop *LoopImpl) Start(p Proc) (procID uint32, doneCh <-chan struct{}) {
	procID, handle := loop.StartLater(p)
	return procID, handle(true)
}

func (loop *LoopImpl) StartLater(proc Proc) (procID uint32, handle func(bool) <-chan struct{}) {
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

	ctx, cancel := context.WithCancel(loop.procCtx)
	pktInCh := make(chan *pb.Packet)
	pktOutCh := make(chan *pb.Packet)

	loop.procData[procID] = loopProcInfo{
		proc:    proc,
		ctx:     ctx,
		cancel:  cancel,
		pktInCh: pktInCh,
	}

	sender := func() {
		logger := loopLogger.NewLogger(fmt.Sprintf("sender[proc=%v]", procID))
		logger.Debug("start")
		defer logger.Debug("done")

		for {
			pkt, ok := <-pktOutCh
			if !ok {
				logger.Debug("proc packet output closed")
				return
			}

			// logger.Debugf("packet: [%v]", pkt)

			// Must be the same.
			pkt.SourceID = procID
			pkt.TargetID = procID

			if !loop.output(pkt) {
				return
			}
		}
	}

	start := func() {
		logger := loopLogger.NewLogger(fmt.Sprintf("proc[%v]", procID))
		logger.Debugf("start kind=%v", proc.Kind())
		defer logger.Debug("done")

		defer loop.Remove(procID)
		defer close(pktOutCh) // must close after proc ends

		loop.procWg.Add(1)
		go func() {
			defer loop.procWg.Done()
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
		proc.Run(&runCtx)
	}

	handle = func(ok bool) <-chan struct{} {
		if !ok {
			logger.Debugf("cancel handle for proc[%v]", procID)
			loop.Remove(procID)
			return nil
		}

		doneCh := make(chan struct{}, 0)

		loop.procWg.Add(1)
		go func() {
			defer loop.procWg.Done()
			defer close(doneCh)
			start()
		}()

		return doneCh
	}

	return procID, handle
}

func (loop *LoopImpl) Remove(procID uint32) bool {
	logger := loopLogger.NewLogger(fmt.Sprintf("remove[%v]", procID))
	// logger.Debug("call")

	loop.procLock.Lock()
	defer loop.procLock.Unlock()

	// logger.Debug("start")
	// defer logger.Debug("end")

	pd, ok := loop.procData[procID]
	if ok {
		logger.Debugf("cleanup")

		pd.cancel()
		close(pd.pktInCh)
		delete(loop.procData, procID)

		// Close remote proc
		go func() {
			pkt := pb.NewCtrlPacket(procID, pb.PacketCtrl_Exit)
			loop.output(pkt)
		}()
	}

	// Shutdown loop if no proc exists
	if len(loop.procData) == 0 {
		logger.Debugf("shutdown loop")
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

		loopLogger.Debug("stop")

		loop.cancel() // shutdown all loops

		for procID := range loop.procData {
			go loop.Remove(procID) // avoid deadlock
		}

		// if conn, ok := loop.ctx.Value("conn").(net.Conn); ok {
		// 	loopLogger.Debug("BreakIO")
		// 	helper.BreakIO(conn)
		// } else {
		// 	panic("cannot find conn in ctx")
		// }
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
		case <-loop.ctx.Done():
			return ErrLoopClosed
		case <-time.NewTimer(loop.dispatchTimeout).C:
			return fmt.Errorf("dispatch timeout")
		}
		return nil
	}

	for pkt := range loop.pktInCh {
		// Handle exit packet from remote loop
		if pkt.SourceID == 0 && pb.IsPacketWithCtrlKind(pkt, pb.PacketCtrl_Exit) {
			logger.Debug("got exit packet:", pkt)
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
			logger.Debug("got error packet [%v]", pkt.String())
			if pkt.SourceID == 0 || pkt.SourceID == pkt.TargetID {
				loop.Remove(pkt.TargetID)
			}
			continue
		}

		err := dispatchToProc(pkt)
		if err != nil {
			if errors.Is(err, ErrLoopClosed) {
				return
			}
			logger.Debugf("dispatch error for [%v]: %v", pkt, err)

			errPkt := &pb.Packet{
				TargetID: pkt.SourceID,
				SourceID: pkt.TargetID,
				Kind:     pb.PacketKind_PacketKindError,
				Data:     []byte(err.Error()),
			}
			if !loop.output(errPkt) {
				return
			}
		}
	}
}

func (loop *LoopImpl) writer() {
	logger := loopLogger.NewLogger("writer")
	logger.Debug("start")
	defer logger.Debug("done")
	defer loop.closer.Close() // close read and write

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
			logger.Debug("loop closed")

			if len(loop.pktOutCh) == 0 {
				return
			}

			logger.Debug("keep flushing")
		}
	}
}

func (loop *LoopImpl) output(pkt *pb.Packet) (ok bool) {
	select {
	case loop.pktOutCh <- pkt:
		return true
	case <-loop.ctx.Done():
		return false
	}
}
