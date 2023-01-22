package worker

import pb "github.com/djosix/med/internal/protobuf"

type Proc interface {
	MsgInCh() chan<- *pb.MedMsg
	Run(loop *LoopImpl, msgOutCh chan<- *pb.MedMsg)
}

type ProcImpl struct {
	msgInCh chan *pb.MedMsg
}

func NewProcImpl(inChSize int) ProcImpl {
	return ProcImpl{
		msgInCh: make(chan *pb.MedMsg, inChSize),
	}
}

func (h *ProcImpl) MsgInCh() chan<- *pb.MedMsg {
	return h.msgInCh
}

func (h *ProcImpl) Run(loop *LoopImpl, msgOutCh chan<- *pb.MedMsg) {
	panic("not implemented")
}
