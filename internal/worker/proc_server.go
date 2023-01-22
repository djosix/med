package worker

import (
	"fmt"

	pb "github.com/djosix/med/internal/protobuf"
)

type ServerProc struct {
	ProcImpl
}

func NewServerProc() *ServerProc {
	return &ServerProc{
		ProcImpl: NewProcImpl(0),
	}
}

func (p *ServerProc) Run(loop *LoopImpl, msgOutCh chan<- *pb.MedMsg) {
	fmt.Println("Run 1")
	msgOutCh <- &pb.MedMsg{
		Type:    pb.MedMsgType_DATA,
		Content: []byte("Message from ServerTermProc"),
	}
	fmt.Println("Run 2")
	fmt.Println(<-p.msgInCh)
	fmt.Println("Run 3")
}
