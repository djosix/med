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
	lastProcID uint32
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
	// frameRw = readwriter.NewSnappyFrameReadWriter(frameRw) // compress frames

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	return &LoopImpl{
		lastProcID: 0,
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
	for _, loopFunc := range []func(){
		loop.loopRead,
		loop.loopDispatch,
		loop.loopWrite,
	} {
		loopFunc := loopFunc
		loop.wg.Add(1)
		go func() {
			defer loop.wg.Done()
			defer loop.Cancel() // stop all other loops after any loop ends
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

	procID = loop.lastProcID + 1
	loop.lastProcID = procID

	ctx, cancel := context.WithCancel(
		context.WithValue(loop.ctx, "procID", procID),
	)

	msgInCh := make(chan *pb.MedMsg)
	msgOutCh := make(chan *pb.MedMsg, 1)

	loop.wg.Add(1)
	go func() {
		defer loop.wg.Done()
		defer logger.Log("done msgOutCh loop of proc", procID)
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
		defer loop.wg.Done()
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
	defer logger.Log("done loopRead")
	for {
		frame, err := loop.frameRw.ReadFrame()
		if err != nil {
			logger.Log("cannot read frame:", err)
			return
		}
		inPkt := pb.MedPkt{}
		if err = proto.Unmarshal(frame, &inPkt); err != nil {
			logger.Log("cannot unmarshal frame to MedPkt:", err)
			continue
		}
		if inPkt.Message.Type == pb.MedMsgType_MedMsgTypeError {
			logger.Log("readLoop:", "got error pkt:", inPkt.String())
			continue
		}
		loop.pktInCh <- &inPkt
	}
}

func (loop *LoopImpl) loopDispatch() {
	defer logger.Log("done loopDispatch")

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
				logger.Log("MedMsg from loop.inPktCh is nil")
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
	defer logger.Log("done loopWrite")
	for {
		select {
		case msg := <-loop.pktOutCh:
			if msg == nil {
				logger.Log("MedPkt from loop.outPktCh is nil")
				return
			}
			buf, err := proto.Marshal(msg)
			if err != nil {
				panic("cannot marshal MedPkt")
			}

			err = loop.frameRw.WriteFrame(buf)
			if err != nil {
				logger.Log(err)
				return
			}
		case <-loop.ctx.Done():
			return
		}
	}
}
