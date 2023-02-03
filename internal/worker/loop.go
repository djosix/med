package worker

import (
	"context"
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

	packetInputCh   chan *pb.Packet
	packetOutputCh  chan *pb.Packet // will have multiple senders so never close it
	frameRW         readwriter.FrameReadWriter
	closer          io.Closer
	dispatchTimeout time.Duration

	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	stopOnce sync.Once

	writeDone    chan struct{}
	dispatchDone chan struct{}
}

type loopProcData struct {
	proc             Proc
	ctx              context.Context
	cancel           context.CancelFunc
	pktInCh          chan *pb.Packet
	closePktInChOnce sync.Once
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

		packetInputCh:   make(chan *pb.Packet, 0),
		packetOutputCh:  make(chan *pb.Packet, 0),
		frameRW:         frameRw,
		closer:          rwc,
		dispatchTimeout: time.Duration(1) * time.Second,

		wg:       sync.WaitGroup{},
		ctx:      loopCtx,
		cancel:   loopCancel,
		stopOnce: sync.Once{},

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

	for _, loopFunc := range []func(){loop.reader, loop.dispatcher, loop.writer} {
		loopFunc := loopFunc
		loop.wg.Add(1)
		go func() {
			defer loop.wg.Done()
			loopFunc()
		}()
	}

	<-loop.ctx.Done()
	logger.Debug("ctx done")

	<-loop.dispatchDone
	logger.Debug("remove all procs")
	for procID := range loop.procData {
		loop.Remove(procID)
	}

	loop.procWg.Wait()
	logger.Debug("all procs done")

	close(loop.packetOutputCh)

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

	pd := &loopProcData{
		proc:    proc,
		ctx:     ctx,
		cancel:  cancel,
		pktInCh: pktInCh,
	}
	loop.procData[procID] = pd

	sender := func() {
		logger := loopLogger.NewLogger(fmt.Sprintf("sender[proc=%v]", procID))
		logger.Debug("start")
		defer logger.Debug("done")

		for pkt := range pktOutCh {

			// logger.Debugf("packet: [%v]", pkt)

			// Must be the same.
			pkt.SourceID = procID
			pkt.TargetID = procID

			select {
			case loop.packetOutputCh <- pkt:
			case <-loop.writeDone:
				return
			}
		}
	}

	start := func() {
		logger := loopLogger.NewLogger(fmt.Sprintf("proc[%v]", procID))
		logger.Debugf("start kind=%v", proc.Kind())
		defer logger.Debug("done")

		sendDone := make(chan struct{})
		go func() {
			sender()
			close(sendDone)
		}()

		runCtx := ProcRunCtx{
			Context:          ctx,
			Cancel:           cancel,
			Loop:             loop,
			PacketInputCh:    pktInCh,
			PacketOutputCh:   pktOutCh,
			packetOutputDone: loop.writeDone,
			ProcID:           procID,
		}
		proc.Run(&runCtx)
		logger.Debug("proc.Run(&runCtx)")
		close(pktOutCh)
		logger.Debug("close(pktOutCh)")
		<-sendDone
		logger.Debug("<-sendDone")
		loop.Remove(procID)
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

		close(pd.pktInCh)
		pd.cancel()
		delete(loop.procData, procID)

		// Close remote proc
		loop.procWg.Add(1)
		go func() {
			defer loop.procWg.Done()
			pkt := pb.NewCtrlPacket(procID, pb.PacketCtrl_Exit)
			loop.putPacketToChannel(pkt, loop.packetOutputCh)
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

// reader exits if reader is closed or loop is done
func (loop *LoopImpl) reader() {
	logger := loopLogger.NewLogger("reader")
	logger.Debug("start")
	defer logger.Debug("done")

	defer close(loop.packetInputCh)

	for loop.isActive() {
		frame, err := loop.frameRW.ReadFrame()
		if err != nil {
			logger.Debug("read frame:", err)
			loop.Stop()
			return
		}

		pkt := &pb.Packet{}
		if err = proto.Unmarshal(frame, pkt); err != nil {
			logger.Error("cannot decode packet:", err)
			continue
		}

		// logger.Debug("pkt in (reader):", pkt)

		loop.putPacketToChannelPassively(pkt, loop.packetInputCh)
	}
}

// dispatcher dispatches the incoming packets to their target processors with timeout.
// It exits if loop.pktInCh is closed or the loop is done.
func (loop *LoopImpl) dispatcher() {
	logger := loopLogger.NewLogger("dispatcher")
	logger.Debug("start")
	defer logger.Debug("done")

	defer close(loop.dispatchDone)

	for {
		pkt := loop.drainPacketFromChannel(loop.packetInputCh)
		if pkt == nil {
			return
		}
		// logger.Debug("pkt in (dispatcher):", pkt)

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

		err := loop.dispatchToProc(pkt)
		// logger.Debug("dispatch to proc:", err)
		if err != nil {
			errPkt := &pb.Packet{
				TargetID: pkt.SourceID,
				SourceID: pkt.TargetID,
				Kind:     pb.PacketKind_PacketKindError,
				Data:     []byte(err.Error()),
			}
			loop.putPacketToChannelPassively(errPkt, loop.packetOutputCh)
		}
	}
}

// writer exits if loop.pktOutCh is closed or the writer fails
func (loop *LoopImpl) writer() {
	logger := loopLogger.NewLogger("writer")
	logger.Debug("start")
	defer logger.Debug("done")

	defer close(loop.writeDone)
	defer loop.closer.Close()

	for pkt := range loop.packetOutputCh {
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
		return fmt.Errorf("proc[%v] not found", pkt.TargetID)
	}

	select {
	case pd.pktInCh <- pkt:
	case <-time.NewTimer(loop.dispatchTimeout).C:
		return fmt.Errorf("dispatch timeout")
	}
	return nil
}

// putPacketToChannel returns false if the packet cannot be put into the channel immediately and the loop context is done
func (loop *LoopImpl) putPacketToChannel(pkt *pb.Packet, ch chan<- *pb.Packet) (ok bool) {
	select {
	case ch <- pkt:
		return true
	default:
		select {
		case ch <- pkt:
			return true
		case <-loop.ctx.Done():
			return false
		}
	}
}

// putPacketToChannelPassively returns false if the packet cannot be put into the channel immediately and the loop context is done
func (loop *LoopImpl) putPacketToChannelPassively(pkt *pb.Packet, ch chan<- *pb.Packet) (ok bool) {
	select {
	case <-loop.ctx.Done():
		return false
	default:
		select {
		case <-loop.ctx.Done():
			return false
		case ch <- pkt:
			return true
		}
	}
}

func (loop *LoopImpl) drainPacketFromChannel(ch <-chan *pb.Packet) *pb.Packet {
	return drainPacketFromChannel(loop.ctx, ch)
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
