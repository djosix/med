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
	Cancel()                  // Get the canceller of ctx
}

type LoopImpl struct {
	nextProcID uint32
	procData   map[uint32]loopProcInfo
	procLock   sync.Mutex
	pktInCh    chan *pb.MedPkt
	pktOutCh   chan *pb.MedPkt
	wg         sync.WaitGroup
	frameRw    readwriter.FrameReadWriter
	ctx        context.Context
	cancel     context.CancelFunc
}

type loopProcInfo struct {
	proc    Proc
	ctx     context.Context
	cancel  context.CancelFunc
	msgInCh chan *pb.MedMsg
}

func NewLoop(ctx context.Context, rw io.ReadWriter) *LoopImpl {
	var frameRw readwriter.FrameReadWriter
	frameRw = readwriter.NewPlainFrameReadWriter(rw)
	frameRw = readwriter.NewSnappyFrameReadWriter(frameRw) // compress frames

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	return &LoopImpl{
		nextProcID: 0,
		procData:   map[uint32]loopProcInfo{},
		procLock:   sync.Mutex{},
		pktInCh:    make(chan *pb.MedPkt),
		pktOutCh:   make(chan *pb.MedPkt),
		wg:         sync.WaitGroup{},
		frameRw:    frameRw,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (loop *LoopImpl) Run() {
	logger := loopLogger.NewLogger("Run")
	logger.Debug("Begin")
	defer logger.Debug("End")

	for _, loopFunc := range []func(){
		loop.loopRead,
		loop.loopDispatch,
		loop.loopWrite,
	} {
		loopFunc := loopFunc
		loop.wg.Add(1)
		go func() {
			defer loop.wg.Done()
			defer loop.Cancel() // stop all other loops after any ends
			loopFunc()
		}()
	}
	loop.wg.Wait()

	if conn, ok := loop.ctx.Value("conn").(net.Conn); ok {
		helper.RefreshIO(conn)
	} else {
		panic("cannot find conn in ctx")
	}
}

func (loop *LoopImpl) Start(p Proc) (procID uint32) {
	loop.procLock.Lock()
	defer loop.procLock.Unlock()

	for {
		procID = loop.nextProcID
		loop.nextProcID++

		_, procExists := loop.procData[procID]
		if !procExists {
			break
		}
	}

	ctx, cancel := context.WithCancel(
		context.WithValue(loop.ctx, "procID", procID),
	)

	msgInCh := make(chan *pb.MedMsg)
	msgOutCh := make(chan *pb.MedMsg, 1)

	loop.wg.Add(1)
	go func() {
		logger := loopLogger.NewLogger(fmt.Sprintf("msgOutCh[%v]", procID))
		logger.Debug("Begin")
		defer loop.wg.Done()
		defer logger.Debug("End")
		for {
			var msg *pb.MedMsg
			select {
			case msg = <-msgOutCh:
				if msg == nil {
					return // msgOutCh is closed
				}
			case <-ctx.Done():
				return
			}
			pkt := &pb.MedPkt{
				SourceID: procID,
				TargetID: procID,
				Message:  msg,
			}
			loop.pktOutCh <- pkt
			// select {
			// case loop.pktOutCh <- pkt:
			// case <-ctx.Done():
			// 	return
			// }
		}
	}()

	loop.procData[procID] = loopProcInfo{
		proc:    p,
		ctx:     ctx,
		cancel:  cancel,
		msgInCh: msgInCh,
	}

	loop.wg.Add(1)
	go func() {
		logger := loopLogger.NewLogger(fmt.Sprintf("proc[%v]", procID))
		logger.Debug("Begin")
		defer loop.wg.Done()
		defer logger.Debug("End")
		runCtx := ProcRunCtx{
			Context:  ctx,
			Cancel:   cancel,
			Loop:     loop,
			MsgInCh:  msgInCh,
			MsgOutCh: msgOutCh,
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
		close(p.msgInCh)
		p.cancel()
	}
	return ok
}

func (loop *LoopImpl) Context() context.Context {
	return loop.ctx
}

func (loop *LoopImpl) Done() <-chan struct{} {
	return loop.ctx.Done()
}

func (loop *LoopImpl) Cancel() {
	loop.cancel()

	if conn, ok := loop.ctx.Value("conn").(net.Conn); ok {
		helper.BreakIO(conn)
	} else {
		panic("cannot find conn in ctx")
	}
}

func (loop *LoopImpl) loopRead() {
	logger := loopLogger.NewLogger("Read")
	logger.Debug("Begin")
	defer logger.Debug("End")

	for {
		frame, err := loop.frameRw.ReadFrame()
		if err != nil {
			logger.Info("cannot read frame:", err)
			return
		}
		inPkt := pb.MedPkt{}
		if err = proto.Unmarshal(frame, &inPkt); err != nil {
			logger.Error("cannot unmarshal frame to MedPkt:", err)
			continue
		}
		if inPkt.Message.Type == pb.MedMsgType_MedMsgTypeError {
			logger.Error("readLoop:", "got error pkt:", inPkt.String())
			continue
		}
		loop.pktInCh <- &inPkt
	}
}

func (loop *LoopImpl) loopDispatch() {
	logger := loopLogger.NewLogger("Dispatch")
	logger.Debug("Begin")
	defer logger.Debug("End")

	dispatchToProc := func(pkt *pb.MedPkt) error {
		loop.procLock.Lock()
		defer loop.procLock.Unlock()

		p, ok := loop.procData[pkt.TargetID]
		if !ok {
			return fmt.Errorf("proc not found with ID %v", pkt.TargetID)
		}

		timeout := time.NewTimer(time.Duration(5) * time.Second)

		select {
		case p.msgInCh <- pkt.Message:
		case <-loop.Done():
			return fmt.Errorf("loop closed")
		case <-timeout.C:
			return fmt.Errorf("dispatch timeout")
		}
		return nil
	}

	for {
		select {
		case inPkt := <-loop.pktInCh:
			if inPkt == nil {
				logger.Error("MedMsg from loop.inPktCh is nil")
				return
			}
			if err := dispatchToProc(inPkt); err != nil {
				loop.pktOutCh <- &pb.MedPkt{
					TargetID: inPkt.SourceID,
					SourceID: inPkt.TargetID,
					Message: &pb.MedMsg{
						Type:    pb.MedMsgType_MedMsgTypeError,
						Content: []byte(err.Error()),
					},
				}
			}
		case <-loop.ctx.Done():
			return
		}
	}
}

func (loop *LoopImpl) loopWrite() {
	logger := loopLogger.NewLogger("Write")
	logger.Debug("Begin")
	defer logger.Debug("End")

	for {
		select {
		case msg := <-loop.pktOutCh:
			if msg == nil {
				logger.Error("MedPkt from loop.outPktCh is nil")
				return
			}
			buf, err := proto.Marshal(msg)
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

// type Logger []string

// func NewLogger() *Logger {

// }
