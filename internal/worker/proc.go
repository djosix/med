package worker

import (
	"context"
	"fmt"

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

func (p *ProcRunCtx) Output(pkt *pb.Packet) bool {
	select {
	case p.PktOutCh <- pkt:
		return true
	case <-p.Loop.Done():
		return false
	}
}

//

func CreateProcClient(kind ProcKind, spec any) (Proc, error) {
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
	case ProcKind_Get:
		if spec, ok := spec.(GetPutSpec); ok {
			proc = NewGetProcClient(spec)
		}
	case ProcKind_Put:
		if spec, ok := spec.(GetPutSpec); ok {
			proc = NewPutProcClient(spec)
		}
	default:
		return nil, fmt.Errorf("proc kind=%v not registered", kind)
	}

	if proc == nil {
		return nil, fmt.Errorf("proc kind=[%v] is nil", kind)
	}

	if proc.Side()&ProcSide_Client == 0 {
		return nil, fmt.Errorf("proc is not for client")
	}

	return proc, nil
}

func CreateProcServer(kind ProcKind) (Proc, error) {
	var proc Proc

	switch kind {
	case ProcKind_Example:
		proc = NewExampleProcServer()
	case ProcKind_Exec:
		proc = NewExecProcServer()
	case ProcKind_Get:
		proc = NewGetProcServer()
	case ProcKind_Put:
		proc = NewPutProcServer()
	default:
		return nil, fmt.Errorf("proc kind=[%v] not registered", kind)
	}

	if proc == nil {
		return nil, fmt.Errorf("proc kind=[%v] is nil", kind)
	}

	if proc.Side()&ProcSide_Server == 0 {
		return nil, fmt.Errorf("proc is not for server")
	}

	return proc, nil
}
