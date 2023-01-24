package worker

import (
	"context"

	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
)

type Proc interface {
	Run(ctx ProcRunCtx)
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

type ProcKind string

const (
	ProcKind_None     ProcKind = "none"
	ProcKind_Example  ProcKind = "example"
	ProcKind_Main     ProcKind = "main"
	ProcKind_Exec     ProcKind = "exec"
	ProcKind_Term     ProcKind = "term"
	ProcKind_LocalPF  ProcKind = "localpf"
	ProcKind_RemotePF ProcKind = "remotepf"
	ProcKind_Upload   ProcKind = "upload"
	ProcKind_Download ProcKind = "download"
)

type ProcSide byte

const (
	ProcSide_None   ProcSide = 0b00
	ProcSide_Client ProcSide = 0b01
	ProcSide_Server ProcSide = 0b10
	ProcSide_Both   ProcSide = 0b11
)

type ProcInfo struct {
	kind ProcKind
	side ProcSide
}

func NewProcInfo(kind ProcKind, side ProcSide) ProcInfo {
	return ProcInfo{kind: kind, side: side}
}

func (p ProcInfo) Kind() ProcKind {
	return p.kind
}

func (p ProcInfo) Side() ProcSide {
	return p.side
}

//

func CreateClientProc(kind ProcKind, spec any) Proc {
	logger := logger.NewLogger("CreateClientProc")

	switch kind {
	case ProcKind_Example:
		if spec, ok := spec.(string); ok {
			return NewExampleProc(spec)
		}

	case ProcKind_Exec:
		if spec, ok := spec.(ProcExecSpec); ok {
			return NewClientExecProc(spec)
		}

	default:
		logger.Error("proc kind=[%v] not registered", kind)
		return nil
	}

	logger.Error("cannot create proc kind=[%v] by spec=%#v", kind, spec)
	return nil
}
