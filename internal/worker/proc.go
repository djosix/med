package worker

import (
	"context"

	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
)

type Proc interface {
	Run(ctx *ProcRunCtx)
	Kind() ProcKind
	Side() ProcSide
}

type ProcRunCtx struct {
	context.Context                    // proc ctx
	Cancel          context.CancelFunc // cancel proc ctx
	Loop            Loop               // the loop running proc
	PktOutCh        chan<- *pb.Packet  // packet output channel
	PktInCh         <-chan *pb.Packet  // packet input channel
	ProcID          uint32
}

//

func CreateProcClient(kind ProcKind, spec any) Proc {
	logger := logger.NewLogger("CreateProcClient")
	var proc Proc

	switch kind {
	case ProcKind_Example:
		if spec, ok := spec.(ExampleSpec); ok {
			proc = NewExampleProcClient(spec)
		}

	case ProcKind_Exec:
		if spec, ok := spec.(ExecSpec); ok {
			proc = NewExecProcClient(spec)
		}

	default:
		logger.Error("proc kind=[%v] not registered", kind)
		return nil
	}

	if proc == nil {
		logger.Error("cannot create proc kind=[%v] by spec=%#v", kind, spec)
		return nil
	}

	if proc.Side()&ProcSide_Client == 0 {
		logger.Error("the created proc is not for client")
		return nil
	}

	return proc
}

func CreateProcServer(kind ProcKind) Proc {
	logger := logger.NewLogger("CreateProcServer")

	var proc Proc

	switch kind {
	case ProcKind_Example:
		proc = NewExampleProcServer()

	case ProcKind_Exec:
		proc = NewExecProcServer()

	default:
		logger.Error("proc kind=[%v] not registered", kind)
	}

	if proc == nil {
		logger.Error("cannot create proc kind=[%v]", kind)
		return nil
	}

	if proc.Side()&ProcSide_Server == 0 {
		logger.Error("the created proc is not for server")
		return nil
	}

	return proc
}
