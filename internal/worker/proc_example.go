package worker

import (
	"fmt"

	pb "github.com/djosix/med/internal/protobuf"
)

type ExampleProc struct {
	ProcImpl
	message string
}

func NewExampleProc(message string) *ExampleProc {
	return &ExampleProc{
		ProcImpl: NewProcImpl(0),
		message:  message,
	}
}

func (p *ExampleProc) Run(loop *LoopImpl, msgOutCh chan<- *pb.MedMsg) {
	fmt.Printf("ExampleProc: send %#v\n", p.message)
	msgOutCh <- &pb.MedMsg{
		Type:    pb.MedMsgType_DATA,
		Content: []byte("Message from ServerTermProc"),
	}

	msg := <-p.msgInCh
	fmt.Printf("ExampleProc: received %#v\n", string(msg.Content))

	loop.Cancel()
}
