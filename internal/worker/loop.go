package worker

import (
	"context"
	"io"
	"log"
	"sync"

	"github.com/djosix/med/internal"
	pb "github.com/djosix/med/internal/protobuf"
	"github.com/djosix/med/internal/readwriter"
	"google.golang.org/protobuf/proto"
)

type Loop interface {
	Run()                  // Run the loop
	Start(h Proc) uint32   // Start a MedProc
	Remove(id uint32) bool // Remove a MedProc
	Done() <-chan struct{} // Get the done chan of ctx
	Cancel()               // Get the canceller of ctx
}

type LoopImpl struct {
	lastProcID uint32
	procs      map[uint32]Proc
	procsMut   sync.Mutex
	pktInCh    chan *pb.MedPkt
	pktOutCh   chan *pb.MedPkt
	wg         sync.WaitGroup
	frameRw    readwriter.FrameReadWriter
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewLoop(ctx context.Context, rw io.ReadWriter) *LoopImpl {
	var frameRw readwriter.FrameReadWriter
	frameRw = readwriter.NewPlainFrameReadWriter(rw)
	frameRw = readwriter.NewSnappyFrameReadWriter(frameRw) // compress frames

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	loop := &LoopImpl{
		lastProcID: 0,
		procs:      map[uint32]Proc{},
		procsMut:   sync.Mutex{},
		pktInCh:    make(chan *pb.MedPkt),
		pktOutCh:   make(chan *pb.MedPkt),
		wg:         sync.WaitGroup{},
		frameRw:    frameRw,
		ctx:        ctx,
		cancel:     cancel,
	}

	return loop
}

func (loop *LoopImpl) createMsgOutCh(procID uint32) chan<- *pb.MedMsg {
	msgOutCh := make(chan *pb.MedMsg, 1)
	loop.wg.Add(1)
	go func() {
		defer loop.wg.Done()
		for {
			var msg *pb.MedMsg

			select {
			case msg = <-msgOutCh:
			case <-loop.ctx.Done():
				return
			}

			pkt := &pb.MedPkt{
				SourceId: procID,
				TargetId: procID,
				Message:  msg,
			}

			select {
			case loop.pktOutCh <- pkt:
			case <-loop.ctx.Done():
				return
			}
		}
	}()
	return msgOutCh
}

func (loop *LoopImpl) Run() {
	loopFuncs := []func(){
		loop.readLoop,
		loop.dispatchLoop,
		loop.writeLoop,
	}
	loop.wg.Add(len(loopFuncs))
	for i := range loopFuncs {
		loopFunc := loopFuncs[i]
		go func() {
			defer loop.wg.Done()
			defer loop.cancel()
			loopFunc()
		}()
	}
	loop.wg.Wait()
}

func (loop *LoopImpl) Done() <-chan struct{} {
	return loop.ctx.Done()
}

func (loop *LoopImpl) Cancel() {
	loop.cancel()
}

func (loop *LoopImpl) Start(p Proc) (procID uint32) {
	loop.procsMut.Lock()
	defer loop.procsMut.Unlock()

	loop.lastProcID++
	procID = loop.lastProcID
	loop.procs[procID] = p

	loop.wg.Add(1)
	go func() {
		p.Run(loop, loop.createMsgOutCh(procID))
		loop.wg.Done()
	}()

	return
}

func (loop *LoopImpl) Remove(procID uint32) (ok bool) {
	loop.procsMut.Lock()
	defer loop.procsMut.Unlock()

	_, ok = loop.procs[procID]
	if !ok {
		return
	}

	delete(loop.procs, procID)
	return
}

func (loop *LoopImpl) readLoop() {
	for {
		frame, err := loop.frameRw.ReadFrame()
		if err != nil {
			log.Println("cannot read frame:", err)
			return
		}
		inPkt := pb.MedPkt{}
		if err = proto.Unmarshal(frame, &inPkt); err != nil {
			log.Println("cannot unmarshal frame to MedPkt:", err)
			continue
		}
		loop.pktInCh <- &inPkt
	}
}

func (loop *LoopImpl) dispatchLoop() {
	for {
		select {
		case inPkt := <-loop.pktInCh:
			if inPkt == nil {
				log.Println("msg from loop.inPktCh is nil")
				return
			}
			if err := loop.dispatchToHandler(inPkt); err != nil {
				loop.pktOutCh <- &pb.MedPkt{
					TargetId: inPkt.SourceId,
					SourceId: inPkt.TargetId,
					Message: &pb.MedMsg{
						Type:    pb.MedMsgType_ERROR,
						Content: []byte(err.Error()),
					},
				}
			}
		case <-loop.ctx.Done():
			return
		}
	}
}

func (loop *LoopImpl) dispatchToHandler(pkt *pb.MedPkt) error {
	loop.procsMut.Lock()
	defer loop.procsMut.Unlock()

	h, ok := loop.procs[pkt.TargetId]
	if !ok {
		return internal.Err
	}

	select {
	case h.MsgInCh() <- pkt.Message:
	case <-loop.Done():
		return internal.Err
	default:
		return internal.Err
	}
	return nil
}

func (loop *LoopImpl) writeLoop() {
	for {
		select {
		case msg := <-loop.pktOutCh:
			if msg == nil {
				log.Println("msg from loop.outPktCh is nil")
				return
			}
			buf, err := proto.Marshal(msg)
			if err != nil {
				panic("cannot marshal MedPkt")
			}

			err = loop.frameRw.WriteFrame(buf)
			if err != nil {
				log.Println(err)
				return
			}
		case <-loop.ctx.Done():
			return
		}
	}
}
