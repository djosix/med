package worker

import (
	"os"

	"github.com/djosix/med/internal/logger"
)

type SelfSpec struct {
	Exit   bool
	Remove bool
}

// Client

type SelfProcClient struct {
	ProcInfo
	spec SelfSpec
}

func NewSelfProcClient(spec SelfSpec) *SelfProcClient {
	return &SelfProcClient{
		ProcInfo: NewProcInfo(ProcKind_Self, ProcSide_Client),
		spec:     spec,
	}
}

func (p *SelfProcClient) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))
	logger.Debugf("spec: %#v", p.spec)

	// Send spec
	SendProcSpec(ctx, p.spec)
}

// Server

type SelfProcServer struct {
	ProcInfo
}

func NewSelfProcServer() *SelfProcServer {
	return &SelfProcServer{
		ProcInfo: NewProcInfo(ProcKind_Self, ProcSide_Server),
	}
}

func (p *SelfProcServer) Run(ctx *ProcRunCtx) {
	logger := logger.NewLogger(string(p.Kind()))

	// Get spec
	spec, err := RecvProcSpec[SelfSpec](ctx)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debug("spec =", spec)

	if spec.Remove {
		if path, err := os.Executable(); err == nil {
			os.Remove(path)
		}
	}

	if spec.Exit {
		os.Exit(1)
	}
}
