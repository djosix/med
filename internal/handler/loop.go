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
	InputMsg(*pb.MedMsg) error
}

type OutMsgFunc = func(*pb.MedMsg) error

type MsgHandlerImpl struct {
	outMsgFunc OutMsgFunc
}

func NewMsgHandlerImpl(outMsgFunc OutMsgFunc) *MsgHandlerImpl {
	return &MsgHandlerImpl{
		outMsgFunc: outMsgFunc,
	}
}

func RunMsgLoop(ctx context.Context, rw io.ReadWriter, msgHandlers map[uint32]MsgHandler) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	loop := &msgLoop{
		lastMsgHandlerID: 0,
		msgHandlers:      msgHandlers,
		msgHandlersMu:    sync.Mutex{},
		inPktCh:          make(chan *pb.MedPkt),
		outPktCh:         make(chan *pb.MedPkt),
		wg:               sync.WaitGroup{},
		f:                readwriter.NewPlainFrameReadWriter(rw),
		ctx:              ctx,
		cancel:           cancel,
	}

	return loop.run()
}

type msgLoop struct {
	lastMsgHandlerID uint32
	msgHandlers      map[uint32]MsgHandler
	msgHandlersMu    sync.Mutex
	inPktCh          chan *pb.MedPkt
	outPktCh         chan *pb.MedPkt
	wg               sync.WaitGroup
	f                readwriter.FrameReadWriter
	ctx              context.Context
	cancel           context.CancelFunc
}

func (loop *msgLoop) createOutMsgFunc(handlerID uint32) OutMsgFunc {
	return func(msg *pb.MedMsg) error {
		pkt := &pb.MedPkt{
			SourceId: handlerID,
			TargetId: handlerID,
			Message:  msg,
		}
		select {
		case loop.outPktCh <- pkt:
			return nil
		default:
			return internal.Err
		}
	}
}

func (loop *msgLoop) createMsgHandler(h MsgHandler) uint32 {
	loop.msgHandlersMu.Lock()
	defer loop.msgHandlersMu.Unlock()

	loop.lastMsgHandlerID++
	newID := loop.lastMsgHandlerID
	loop.msgHandlers[newID] = h
	return newID
}

func (loop *msgLoop) deleteMsgHandler(id uint32) bool {
	loop.msgHandlersMu.Lock()
	defer loop.msgHandlersMu.Unlock()

	if _, ok := loop.msgHandlers[id]; ok {
		delete(loop.msgHandlers, id)
		return true
	}
	return false
}

func (loop *msgLoop) dispatchToHandler(pkt *pb.MedPkt) error {
	loop.msgHandlersMu.Lock()
	defer loop.msgHandlersMu.Unlock()

	h, ok := loop.msgHandlers[pkt.TargetId]
	if !ok {
		return internal.Err
	}

	return h.InputMsg(pkt.Message)
}

func (loop *msgLoop) run() error {
	loop.wg.Add(3)
	go loop.readMsgLoop()
	go loop.handleMsgLoop()
	go loop.writeMsgLoop()
	loop.wg.Wait()
	return nil
}

func (loop *msgLoop) readMsgLoop() {
	for {
		frame, err := loop.f.ReadFrame()
		if err != nil {
			log.Println("cannot read frame:", err)
			break
		}
		inPkt := pb.MedPkt{}
		if err = proto.Unmarshal(frame, &inPkt); err != nil {
			log.Println("cannot unmarshal frame to MedPkt:", err)
			continue
		}
		loop.inPktCh <- &inPkt
	}
	loop.wg.Done()
	loop.close()
}

func (loop *msgLoop) handleMsgLoop() {
	for done := false; !done; {
		select {
		case inPkt := <-loop.inPktCh:
			if inPkt == nil {
				log.Println("msg from loop.inPktCh is nil")
				done = true
				break
			}
			if err := loop.dispatchToHandler(inPkt); err != nil {
				loop.outPktCh <- &pb.MedPkt{
					TargetId: inPkt.SourceId,
					SourceId: inPkt.TargetId,
					Message: &pb.MedMsg{
						Type:    pb.MedMsgType_ERROR,
						Content: []byte(err.Error()),
					},
				}
			}
		case <-loop.ctx.Done():
			done = true
		}
	}
	loop.wg.Done()
	loop.close()
}

func (loop *msgLoop) writeMsgLoop() {
	for done := false; !done; {
		select {
		case msg := <-loop.outPktCh:
			if msg == nil {
				log.Println("msg from loop.outPktCh is nil")
				done = true
				break
			}
			buf, err := proto.Marshal(msg)
			if err != nil {
				panic("cannot marshal MedPkt")
			}

			err = loop.f.WriteFrame(buf)
			if err != nil {
				log.Println(err)
				done = true
				break
			}
		case <-loop.ctx.Done():
			done = true
		}
	}
	loop.wg.Done()
	loop.close()
}

func (loop *msgLoop) close() {
	loop.cancel()
	close(loop.inPktCh)
	close(loop.outPktCh)
}
