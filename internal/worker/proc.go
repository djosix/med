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
	ProcID          uint32
	pktOutCh        chan<- *pb.Packet // packet output channel
	pktOutDone      <-chan struct{}
	pktInCh         <-chan *pb.Packet // packet input channel
}

func (p *ProcRunCtx) PacketOutput(pkt *pb.Packet) bool {
	select {
	case p.pktOutCh <- pkt:
		return true
	case <-p.pktOutDone:
		return false
	}
}

func (p *ProcRunCtx) PacketInput() *pb.Packet {
	return p.PacketInputWithDone(p.Done())
}

func (p *ProcRunCtx) PacketInputWithDone(done <-chan struct{}) *pb.Packet {
	select {
	case pkt := <-p.pktInCh:
		return pkt
	case <-done:
		return nil
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
