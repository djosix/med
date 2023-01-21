package handler

import (
	"github.com/djosix/med/internal"
	pb "github.com/djosix/med/internal/protobuf"
)

type ClientMsgHandler struct {
	msgInCh chan *pb.MedMsg
}

func (h *ClientMsgHandler) InputMsg(msg *pb.MedMsg) error {
	select {
	case h.msgInCh <- msg:
		return nil
	default:
		return internal.Err
	}
}

func (h *ClientMsgHandler) Run(loop *MsgLoop, msgOutCh chan<- *pb.MedMsg) {

}
