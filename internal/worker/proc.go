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
	PacketOutputCh  chan<- *pb.Packet // packet output channel
	pktOutDone      <-chan struct{}
	PacketInputCh   <-chan *pb.Packet // packet input channel
}

func (ctx *ProcRunCtx) OutputPacket(pkt *pb.Packet) bool {
	select {
	case ctx.PacketOutputCh <- pkt:
		return true
	case <-ctx.pktOutDone:
		return false
	}
}

func (ctx *ProcRunCtx) InputPacket() *pb.Packet {
	return ctx.InputPacketWithDone(ctx.Done())
}

func (ctx *ProcRunCtx) InputPacketWithDone(doneCh <-chan struct{}) *pb.Packet {
	select {
	case pkt := <-ctx.PacketInputCh:
		return pkt
	case <-doneCh:
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
	case ProcKind_Self:
		if spec, ok := spec.(SelfSpec); ok {
			proc = NewSelfProcClient(spec)
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
	case ProcKind_Self:
		proc = NewSelfProcServer()
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
