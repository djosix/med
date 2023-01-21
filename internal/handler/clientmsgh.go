package handler

import (
	pb "github.com/djosix/med/internal/protobuf"
)

type MsgHandlerImpl struct {
	msgInCh chan *pb.MedMsg
}

func (h *MsgHandlerImpl) MsgInCh(msg *pb.MedMsg) chan<- *pb.MedMsg {
	return h.msgInCh
}

func (h *MsgHandlerImpl) RunLoop(loop *MsgLoop, msgOutCh chan<- *pb.MedMsg) {
	panic("not implemented")
}

type ClientMsgHandler struct {
	MsgHandlerImpl
}

func (h *ClientMsgHandler) RunLoop(loop *MsgLoop, msgOutCh chan<- *pb.MedMsg) {

}
