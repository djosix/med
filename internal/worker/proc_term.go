package worker

import (
	"fmt"

	pb "github.com/djosix/med/internal/protobuf"
)

type ClientTermProc struct{}

func NewClientTermProc() *ClientTermProc {
	return &ClientTermProc{}
}

func (p *ClientTermProc) Run(ctx ProcRunCtx) {
	fmt.Println("Run 1")
	ctx.MsgOutCh <- &pb.MedMsg{
		Type:    pb.MedMsgType_MedMsgTypeData,
		Content: []byte("Message from ClientTermProc"),
	}
	fmt.Println("Run 2")
	fmt.Println(<-ctx.MsgInCh)
	fmt.Println("Run 3")
}

type ServerTermProc struct{}

func NewServerTermProc() *ServerTermProc {
	return &ServerTermProc{}
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
