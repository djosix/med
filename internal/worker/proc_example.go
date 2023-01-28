package worker

import (
	"fmt"

	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
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
	logger := logger.NewLogger(string(p.Kind()))

	// Send spec
	SendProcSpec(ctx, p.spec)
	ctx.pktOutCh <- helper.NewDataPacket([]byte(fmt.Sprintf("Hello %s from client", p.spec.Name)))
	logger.Info(string((<-ctx.pktInCh).Data))
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
	logger := logger.NewLogger(string(p.Kind()))

	// Get spec
	spec, err := RecvProcSpec[ExampleSpec](ctx)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debug("spec =", spec)

	ctx.pktOutCh <- helper.NewDataPacket([]byte(fmt.Sprintf("Hello %s from server", spec.Name)))
	logger.Info(string((<-ctx.pktInCh).Data))
}
