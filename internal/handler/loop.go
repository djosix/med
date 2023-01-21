package handler

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

type MsgHandler interface {
	MsgInCh() chan<- *pb.MedMsg
	RunLoop(loop *MsgLoop, msgOutCh chan<- *pb.MedMsg)
}

type MsgLoop struct {
	lastMsgHandlerID uint32
	msgHandlers      map[uint32]MsgHandler
	msgHandlersMu    sync.Mutex
	pktInCh          chan *pb.MedPkt
	pktOutCh         chan *pb.MedPkt
	wg               sync.WaitGroup
	f                readwriter.FrameReadWriter
	ctx              context.Context
	cancel           context.CancelFunc
}

func RunMsgLoop(ctx context.Context, rw io.ReadWriter, msgHandlers map[uint32]MsgHandler) error {
	if len(msgHandlers) == 0 {
		panic("must have at least one msgHandler")
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	loop := &MsgLoop{
		lastMsgHandlerID: 0,
		msgHandlers:      msgHandlers,
		msgHandlersMu:    sync.Mutex{},
		pktInCh:          make(chan *pb.MedPkt),
		pktOutCh:         make(chan *pb.MedPkt),
		wg:               sync.WaitGroup{},
		f:                readwriter.NewPlainFrameReadWriter(rw),
		ctx:              ctx,
		cancel:           cancel,
	}

	for id, h := range msgHandlers {
		loop.wg.Add(1)
		go func(h MsgHandler, id uint32) {
			defer loop.wg.Done()
			h.RunLoop(loop, loop.createMsgOutCh(id))
		}(h, id)
	}

	return loop.runAllLoops()
}

func (loop *MsgLoop) createMsgOutCh(handlerID uint32) chan<- *pb.MedMsg {
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
				SourceId: handlerID,
				TargetId: handlerID,
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

func (loop *MsgLoop) CreateMsgHandler(h MsgHandler) uint32 {
	loop.msgHandlersMu.Lock()
	defer loop.msgHandlersMu.Unlock()

	loop.lastMsgHandlerID++
	newID := loop.lastMsgHandlerID
	loop.msgHandlers[newID] = h
	return newID
}

func (loop *MsgLoop) DeleteMsgHandler(id uint32) bool {
	loop.msgHandlersMu.Lock()
	defer loop.msgHandlersMu.Unlock()

	if _, ok := loop.msgHandlers[id]; ok {
		delete(loop.msgHandlers, id)
		return true
	}
	return false
}

func (loop *MsgLoop) Done() <-chan struct{} {
	return loop.ctx.Done()
}

func (loop *MsgLoop) Cancel() {
	loop.cancel()
}

func (loop *MsgLoop) runAllLoops() error {
	loopFuncs := []func(){
		loop.readLoop,
		loop.dispatchLoop,
		loop.writeLoop,
	}
	for i := range loopFuncs {
		loopFunc := loopFuncs[i]
		loop.wg.Add(1)
		go func() {
			defer loop.wg.Done()
			defer loop.cancel()
			loopFunc()
		}()
	}
	loop.wg.Wait()
	return nil
}

func (loop *MsgLoop) readLoop() {
	for {
		frame, err := loop.f.ReadFrame()
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

func (loop *MsgLoop) dispatchLoop() {
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

func (loop *MsgLoop) dispatchToHandler(pkt *pb.MedPkt) error {
	loop.msgHandlersMu.Lock()
	defer loop.msgHandlersMu.Unlock()

	h, ok := loop.msgHandlers[pkt.TargetId]
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

func (loop *MsgLoop) writeLoop() {
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

			err = loop.f.WriteFrame(buf)
			if err != nil {
				log.Println(err)
				return
			}
		case <-loop.ctx.Done():
			return
		}
	}
}
