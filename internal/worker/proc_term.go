package worker

import (
	"fmt"

	pb "github.com/djosix/med/internal/protobuf"
)

type ClientTermProc struct {
	ProcImpl
}

func NewClientTermProc() *ClientTermProc {
	return &ClientTermProc{
		ProcImpl: NewProcImpl(0),
	}
}

func (p *ClientTermProc) Run(loop *LoopImpl, msgOutCh chan<- *pb.MedMsg) {
	fmt.Println("Run 1")
	msgOutCh <- &pb.MedMsg{
		Type:    pb.MedMsgType_DATA,
		Content: []byte("Message from ClientTermProc"),
	}
	fmt.Println("Run 2")
	fmt.Println(<-p.msgInCh)
	fmt.Println("Run 3")
}

type ServerTermProc struct {
	ProcImpl
}

func NewServerTermProc() *ServerTermProc {
	return &ServerTermProc{
		ProcImpl: NewProcImpl(0),
	}
}

func (p *ServerTermProc) Run(loop *LoopImpl, msgOutCh chan<- *pb.MedMsg) {
	// fmt.Println("Run 1")
	// msgOutCh <- &pb.MedMsg{
	// 	Type:    pb.MedMsgType_DATA,
	// 	Content: []byte("Message from ServerTermProc"),
	// }

	// msg := <-p.msgInCh
	// fmt.Println("Run 2")
	// fmt.Println()
	// fmt.Println("Run 3")
}
