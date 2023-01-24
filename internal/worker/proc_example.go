package worker

import (
	"fmt"

	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
)

type ExampleSpec struct {
	Name string
}

// Client

type ExampleProcClient struct {
	ProcInfo
	spec ExampleSpec
}

func NewExampleProcClient(spec ExampleSpec) *ExampleProcClient {
	return &ExampleProcClient{
		ProcInfo: NewProcInfo(ProcKind_Example, ProcSide_Client),
		spec:     spec,
	}
}

func (p *ExampleProcClient) Run(ctx *ProcRunCtx) {
	// Send spec
	SendProcSpec(ctx, p.spec)

	ctx.PktOutCh <- &pb.Packet{
		Kind: pb.PacketKind_PacketKindData,
		Data: []byte(fmt.Sprintf("Hello %s from client", p.spec.Name)),
	}

	logger.Info(string((<-ctx.PktInCh).Data))
}

// Server

type ExampleProcServer struct {
	ProcInfo
}

func NewExampleProcServer() *ExampleProcServer {
	return &ExampleProcServer{
		ProcInfo: NewProcInfo(ProcKind_Example, ProcSide_Server),
	}
}

func (p *ExampleProcServer) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger("ExampleProcServer")

	// Get spec
	spec, err := RecvProcSpec[ExampleSpec](ctx)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debug("spec =", spec)

	ctx.PktOutCh <- &pb.Packet{
		Kind: pb.PacketKind_PacketKindData,
		Data: []byte(fmt.Sprintf("Hello %s from server", spec.Name)),
	}

	logger.Info(string((<-ctx.PktInCh).Data))
}
