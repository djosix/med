package worker

import (
	"fmt"

	pb "github.com/djosix/med/internal/protobuf"
)

type ExampleProc struct {
	message string
}

func NewExampleProc(message string) *ExampleProc {
	return &ExampleProc{
		message: message,
	}
}

func (p *ExampleProc) Run(ctx ProcRunCtx) {
	fmt.Printf("ExampleProc: sending %#v\n", p.message)
	ctx.MsgOutCh <- &pb.MedMsg{
		Type: pb.MedMsgType_MedMsgTypeData,
		Data: []byte(p.message),
	}
	fmt.Printf("ExampleProc: waiting for response\n")

	msg := <-ctx.MsgInCh
	fmt.Printf("ExampleProc: received %#v\n", string(msg.Data))

	ctx.Loop.Cancel()
}
