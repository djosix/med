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
	ErrLoopClosed = fmt.Errorf("loop closed")
	ErrTimeout    = fmt.Errorf("timeout")
	loopLogger    = logger.NewLogger("Loop")
)

type Loop interface {
	Run()                     // Run the loop
	Start(h Proc) uint32      // Start a MedProc
	Remove(id uint32) bool    // Remove a MedProc
	Done() <-chan struct{}    // Get the done chan of ctx
	Context() context.Context // Get the ctx of loop
	Stop()                    // Stop it
}

type LoopImpl struct {
	lastProcID uint32
	procData   map[uint32]loopProcInfo
	procLock   sync.Mutex
	runLock    sync.Mutex

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
	ctx, cancel = context.WithCancel(ctx)

	return &LoopImpl{
		lastProcID: 0,
		procData:   map[uint32]loopProcInfo{},
		procLock:   sync.Mutex{},
		runLock:    sync.Mutex{},

		pktInCh:         make(chan *pb.Packet),
		pktOutCh:        make(chan *pb.Packet),
		frameRw:         frameRw,
		dispatchTimeout: time.Duration(5) * time.Second,

		wg:       sync.WaitGroup{},
		ctx:      ctx,
		cancel:   cancel,
		stopOnce: sync.Once{},
	}
}

func (loop *LoopImpl) Run() {
	loop.runLock.Lock()
	defer loop.runLock.Unlock()

	logger := loopLogger.NewLogger("Run")
	logger.Debug("start")
	defer logger.Debug("done")

	loop.stopOnce = sync.Once{}

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
		logger.Debug("RefreshIO(conn)")
		helper.RefreshIO(conn)
	} else {
		panic("cannot find conn in ctx")
	}
}

func (loop *LoopImpl) Start(p Proc) (procID uint32) {
	loop.procLock.Lock()
	defer loop.procLock.Unlock()

	// get a valid procID
	{
		procID = loop.lastProcID + 1
		for _, exists := loop.procData[procID]; exists || procID == 0; {
			procID++
		}
		loop.lastProcID = procID
	}

	ctx, cancel := context.WithCancel(loop.ctx)
	pktInCh := make(chan *pb.Packet)
	pktOutCh := make(chan *pb.Packet, 1)

	loop.wg.Add(1)
	go func() {
		logger := loopLogger.NewLogger(fmt.Sprintf("pktOutCh[%v]", procID))
		logger.Debug("start")

		defer loop.wg.Done()
		defer logger.Debug("done")

		for {
			var pkt *pb.Packet
			select {
			case pkt = <-pktOutCh:
				if pkt == nil {
					return // pktOutCh is closed
				}
			case <-ctx.Done():
				return
			}

			pkt.SourceID = procID
			pkt.TargetID = procID

			select {
			case loop.pktOutCh <- pkt:
			case <-ctx.Done():
				return
			}
		}
	}()

	loop.procData[procID] = loopProcInfo{
		proc:    p,
		ctx:     ctx,
		cancel:  cancel,
		pktInCh: pktInCh,
	}

	loop.wg.Add(1)
	go func() {
		logger := loopLogger.NewLogger(fmt.Sprintf("proc[%v]", procID))
		logger.Debug("start")

		defer loop.wg.Done()
		defer logger.Debug("done")
		defer loop.Remove(procID)

		runCtx := ProcRunCtx{
			Context:  ctx,
			Cancel:   cancel,
			Loop:     loop,
			PktInCh:  pktInCh,
			PktOutCh: pktOutCh,
			ProcID:   procID,
		}
		p.Run(runCtx)
	}()

	return
}

func (loop *LoopImpl) Remove(procID uint32) bool {
	loop.procLock.Lock()
	defer loop.procLock.Unlock()

	p, ok := loop.procData[procID]
	if ok {
		delete(loop.procData, procID)
		close(p.pktInCh)
		p.cancel()
	}

	// shutdown loop if no proc exists
	if len(loop.procData) == 0 {
		loop.Stop()
	}

	return ok
}

func (loop *LoopImpl) Context() context.Context {
	return loop.ctx
}

func (loop *LoopImpl) Done() <-chan struct{} {
	return loop.ctx.Done()
}

func (loop *LoopImpl) Stop() {
	loop.stopOnce.Do(func() {
		logger := loopLogger.NewLogger("Cancel")
		logger.Debug("cancel()")
		loop.cancel()

		if conn, ok := loop.ctx.Value("conn").(net.Conn); ok {
			logger.Debug("BreakIO(conn)")
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

	for {
		frame, err := loop.frameRw.ReadFrame()
		if err != nil {
			logger.Debug("ReadFrame:", err)
			return
		}
		pkt := pb.Packet{}
		if err = proto.Unmarshal(frame, &pkt); err != nil {
			logger.Warn("Unmarshal(frame, pb.Packet):", err)
			continue
		}
		if pkt.Kind == pb.PacketKind_PacketKindError {
			logger.Error("got error pkt:", pkt.String())
			continue
		}
		loop.pktInCh <- &pkt
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

		p, ok := loop.procData[pkt.TargetID]
		if !ok {
			return fmt.Errorf("proc[%v] not found", pkt.TargetID)
		}

		timeout := time.NewTimer(loop.dispatchTimeout)

		select {
		case p.pktInCh <- pkt:
		case <-loop.Done():
			return fmt.Errorf("loop closed")
		case <-timeout.C:
			return fmt.Errorf("dispatch timeout")
		}
		return nil
	}

	for {
		select {
		case pkt := <-loop.pktInCh:
			if pkt == nil {
				logger.Error("loop.inPktCh: nil")
				return
			}
			if err := dispatchToProc(pkt); err != nil {
				loop.pktOutCh <- &pb.Packet{
					TargetID: pkt.SourceID,
					SourceID: pkt.TargetID,
					Kind:     pb.PacketKind_PacketKindError,
					Data:     []byte(err.Error()),
				}
			}
		case <-loop.ctx.Done():
			return
		}
	}
}

func (loop *LoopImpl) writer() {
	logger := loopLogger.NewLogger("writer")
	logger.Debug("start")
	defer logger.Debug("done")

	for {
		select {
		case pkt := <-loop.pktOutCh:
			if pkt == nil {
				logger.Error("MedPkt from loop.outPktCh is nil")
				return
			}
			buf, err := proto.Marshal(pkt)
			if err != nil {
				panic("cannot marshal MedPkt")
			}

			err = loop.frameRw.WriteFrame(buf)
			if err != nil {
				logger.Error("WriteFrame:", err)
				return
			}
		case <-loop.ctx.Done():
			return
		}
	}
}
