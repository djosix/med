package worker

import (
	"fmt"

	pb "github.com/djosix/med/internal/protobuf"
)

type ExampleProc struct {
	ProcInfo
	message string
}

func NewExampleProc(message string) *ExampleProc {
	return &ExampleProc{
		ProcInfo: NewProcInfo(ProcKind_Example, ProcSide_Both),
		message:  message,
	}
}

func (p *ExampleProc) Run(ctx ProcRunCtx) {
	fmt.Printf("ExampleProc: sending %#v\n", p.message)
	ctx.PktOutCh <- &pb.Packet{
		Kind: pb.PacketKind_PacketKindData,
		Data: []byte(p.message),
	}
	fmt.Printf("ExampleProc: waiting for response\n")

	pkt := <-ctx.PktInCh
	fmt.Printf("ExampleProc: received %#v\n", string(pkt.Data))
}
