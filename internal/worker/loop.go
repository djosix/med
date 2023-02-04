package worker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
	"github.com/djosix/med/internal/readwriter"
	"google.golang.org/protobuf/proto"
)

var (
	loopLogger = logger.NewLogger("loop")

	ErrLoopClosed = fmt.Errorf("loop closed")
)

type Loop interface {
	Run()                                                 // Run the loop
	Start(h Proc) (procID uint32, doneCh <-chan struct{}) // Start a Proc
	StartLater(p Proc) (
		procID uint32,
		handle func(bool) (doneCh <-chan struct{}),
	) // Start a Proc later
	Remove(id uint32) bool // Remove a Proc
	Done() <-chan struct{} // Get the done chan of loop ctx
	Stop()                 // Stop the loop
}

type LoopImpl struct {
	lastProcID uint32
	procData   map[uint32]*loopProcData
	procLock   sync.Mutex
	procCtx    context.Context
	procCancel context.CancelFunc
	procWg     sync.WaitGroup

	runLock sync.Mutex

	pktInCh    chan *pb.Packet // sender: reader
	pktOutCh   chan *pb.Packet // sender: dispatcher, processor's senders
	pktOutChWg sync.WaitGroup  // sender wait group

	frameRW         readwriter.FrameReadWriter
	closer          io.Closer
	dispatchTimeout time.Duration

	wg       *helper.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once

	writeDone    chan struct{}
	dispatchDone chan struct{}
}

type loopProcData struct {
	proc         Proc
	ctx          context.Context
	cancel       context.CancelFunc
	pktInCh      chan *pb.Packet
	closePktInCh func()
}

// func (pd *loopProcData) closeInput() {
// 	pd.closePktInChOnce.Do(func() {
// 		close(pd.pktInCh)
// 	})
// }

func NewLoop(ctx context.Context, rwc io.ReadWriteCloser) *LoopImpl {
	var frameRw readwriter.FrameReadWriter
	frameRw = readwriter.NewPlainFrameReadWriter(rwc)
	frameRw = readwriter.NewSnappyFrameReadWriter(frameRw) // compress frames

	loopCtx, loopCancel := context.WithCancel(ctx)
	procCtx, procCancel := context.WithCancel(ctx)

	return &LoopImpl{
		lastProcID: 0,
		procData:   map[uint32]*loopProcData{},
		procLock:   sync.Mutex{},
		procCtx:    procCtx,
		procCancel: procCancel,
		procWg:     sync.WaitGroup{},

		runLock: sync.Mutex{},

		pktInCh:         make(chan *pb.Packet, 0),
		pktOutCh:        make(chan *pb.Packet, 0),
		frameRW:         frameRw,
		closer:          rwc,
		dispatchTimeout: time.Duration(1) * time.Second,

		wg:       helper.NewWaitGroup(),
		ctx:      loopCtx,
		cancel:   loopCancel,
		stopOnce: sync.Once{},

		pktOutChWg: sync.WaitGroup{},

		writeDone:    make(chan struct{}),
		dispatchDone: make(chan struct{}),
	}
}

func (loop *LoopImpl) Run() {
	loop.runLock.Lock()
	defer loop.runLock.Unlock()

	logger := loopLogger.NewLogger("run")
	logger.Debug("start")
	defer logger.Debug("done")

	if !loop.isActive() {
		logger.Error("cannot run a closed loop")
		return
	}

	loop.wg.DoneFn = func(name string) {
		logger.Debugf("wg: done=%v active=%v", name, loop.wg.ActiveNames())
	}

	loop.wg.GoName("reader", func() {
		defer close(loop.pktInCh)
		loop.reader()
	})

	loop.pktOutChWg.Add(1)
	loop.wg.GoName("dispatcher", func() {
		defer func() {
			loop.pktOutChWg.Done()
			close(loop.dispatchDone)
		}()
		loop.dispatcher()
	})

	loop.wg.GoName("writer", func() {
		defer func() {
			close(loop.writeDone)
			loop.closer.Close() // ensure IO is closed
		}()
		loop.writer()
	})

	<-loop.ctx.Done()
	logger.Debug("loop.ctx is done")

	<-loop.dispatchDone
	logger.Debug("remove all procs")
	for procID := range loop.procData {
		loop.Remove(procID)
	}

	loop.procWg.Wait()
	logger.Debug("all procs done")

	close(loop.pktOutCh)

	loop.wg.Wait()
	logger.Debug("all loops done")
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
	closePktInChOnce := sync.Once{}
	closePktInCh := func() {
		closePktInChOnce.Do(func() { close(pktInCh) })
	}
	loop.procData[procID] = &loopProcData{
		proc:         proc,
		ctx:          ctx,
		cancel:       cancel,
		pktInCh:      pktInCh,
		closePktInCh: closePktInCh,
	}

	logger := loopLogger.NewLoggerf("proc-%v", procID)

	loop.wg.GoName(fmt.Sprintf("closePktInCh-%v", procID), func() {
		select {
		case <-ctx.Done():
		case <-loop.dispatchDone:
			closePktInCh()
		}
	})

	sender := func() {
		logger := logger.NewLogger("sender")
		logger.Debug("start")
		defer logger.Debug("done")

		for pkt := range pktOutCh {
			// logger.Debugf("packet: [%v]", pkt)

			// Must be the same.
			pkt.SourceID = procID
			pkt.TargetID = procID

			if ok := loop.outputPacket(pkt); !ok {
				return
			}
		}
	}

	start := func() {
		logger.Debugf("start kind=%v", proc.Kind())
		defer logger.Debug("done")

		sendDone := make(chan struct{})
		loop.pktOutChWg.Add(1)
		go func() {
			sender()
			close(sendDone)
			loop.pktOutChWg.Done()
		}()

		runCtx := ProcRunCtx{
			Context:    ctx,
			Cancel:     cancel,
			Loop:       loop,
			PktInCh:    pktInCh,
			PktOutCh:   pktOutCh,
			PktOutDone: loop.writeDone,
			ProcID:     procID,
		}
		proc.Run(&runCtx)

		// It's ok to close pktOutCh because proc is the only sender,
		// and we need sender to loop.pktOutCh to stop.
		close(pktOutCh)
		<-sendDone

		loop.Remove(procID)
	}

	handle = func(ok bool) <-chan struct{} {
		if !ok {
			logger.Debugf("cancel handle for proc-%v", procID)
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
	logger := loopLogger.NewLoggerf("remove-%v", procID)

	loop.procLock.Lock()
	defer loop.procLock.Unlock()

	pd, ok := loop.procData[procID]
	if ok {
		logger.Debugf("cleanup")

		pd.closePktInCh()
		pd.cancel()
		delete(loop.procData, procID)

		// Close remote proc
		loop.procWg.Add(1)
		go func() {
			defer loop.procWg.Done()
			pkt := pb.NewCtrlPacket(procID, pb.PacketCtrl_Exit)
			loop.outputPacket(pkt)
		}()
	}

	// Shutdown loop if no proc exists
	if len(loop.procData) == 0 {
		logger.Debugf("shutdown loop")
		loop.Stop()
	}

	return ok
}

func (loop *LoopImpl) Done() <-chan struct{} {
	return loop.ctx.Done()
}

func (loop *LoopImpl) Stop() {
	loopLogger.Debug("stop")
	loop.cancel()
}

// reader reads frames, decodes them as packets, and send packets to loop.pktInCh.
// It exits only when the frame reader fails to read a frame.
func (loop *LoopImpl) reader() {
	logger := loopLogger.NewLogger("reader")
	logger.Debug("start")
	defer logger.Debug("done")

	for loop.isActive() {
		frame, err := loop.frameRW.ReadFrame()
		if err != nil {
			logger.Debug("read frame:", err)
			return
		}

		pkt := &pb.Packet{}
		if err = proto.Unmarshal(frame, pkt); err != nil {
			logger.Error("cannot decode packet:", err)
			continue
		}

		select {
		case loop.pktInCh <- pkt:
		case <-loop.ctx.Done():
			return
		}
	}
}

// dispatcher gets packets from loop.pktInCh and dispatches the packets to their target processors' pktInCh with timeout.
// It exits if loop.pktInCh is closed or the loop is done.
func (loop *LoopImpl) dispatcher() {
	logger := loopLogger.NewLogger("dispatcher")
	logger.Debug("start")
	defer logger.Debug("done")

	getNextPacket := func() *pb.Packet {
		select {
		case pkt := <-loop.pktInCh:
			return pkt
		case <-loop.ctx.Done():
			return nil
		}
	}

	for loop.isActive() {
		pkt := getNextPacket()
		if pkt == nil {
			return
		}

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
			logger.Errorf("remote proc-%v error: %v", pkt.SourceID, string(pkt.Data))
			logger.Debug("got error packet [%v]", pkt.String())
			if pkt.SourceID == 0 || pkt.SourceID == pkt.TargetID {
				loop.Remove(pkt.TargetID)
			}
			continue
		}

		err := loop.dispatchToProc(pkt)
		// logger.Debug("dispatch to proc:", err)
		if err != nil {
			errPkt := &pb.Packet{
				TargetID: pkt.SourceID,
				SourceID: pkt.TargetID,
				Kind:     pb.PacketKind_PacketKindError,
				Data:     []byte(err.Error()),
			}
			loop.outputPacket(errPkt)
		}
	}
}

// writer exits if loop.pktOutCh is closed or the writer fails.
func (loop *LoopImpl) writer() {
	logger := loopLogger.NewLogger("writer")
	logger.Debug("start")
	defer logger.Debug("done")

	for pkt := range loop.pktOutCh {
		// logger.Debug("pkt out:", pkt)

		buf, err := proto.Marshal(pkt)
		if err != nil {
			panic("cannot encode packet")
		}

		err = loop.frameRW.WriteFrame(buf)
		if err != nil {
			logger.Debug("write frame:", err)
			return
		}
	}
}

func (loop *LoopImpl) dispatchToProc(pkt *pb.Packet) error {
	loop.procLock.Lock()
	defer loop.procLock.Unlock()

	// logger.Debugf("packet: [%v]", pkt)
	pd, ok := loop.procData[pkt.TargetID]
	if !ok {
		return fmt.Errorf("proc-%v not found", pkt.TargetID)
	}

	select {
	case pd.pktInCh <- pkt:
	case <-time.NewTimer(loop.dispatchTimeout).C:
		return fmt.Errorf("dispatch timeout")
	}
	return nil
}

func (loop *LoopImpl) outputPacket(pkt *pb.Packet) (ok bool) {
	// If loop.writer is done, no one will drain loop.pktOutCh,
	// so we need to stop sending pkt towards it.
	select {
	case loop.pktOutCh <- pkt:
		return true
	case <-loop.writeDone:
		return false
	}
}

func drainPacketFromChannel(ctx context.Context, ch <-chan *pb.Packet) *pb.Packet {
	select {
	case pkt := <-ch:
		return pkt
	default:
		select {
		case pkt := <-ch:
			return pkt
		case <-ctx.Done():
			return nil
		}
	}
}

func (loop *LoopImpl) isActive() bool {
	select {
	case <-loop.ctx.Done():
		return false
	default:
		return true
	}
}
