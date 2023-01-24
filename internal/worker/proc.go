package worker

import (
	"context"

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

type ProcKind byte

const (
	ProcKind_None     ProcKind = 0
	ProcKind_Example  ProcKind = 1
	ProcKind_Exec     ProcKind = 11
	ProcKind_Term     ProcKind = 12
	ProcKind_LocalPF  ProcKind = 21
	ProcKind_RemotePF ProcKind = 22
	ProcKind_Upload   ProcKind = 31
	ProcKind_Download ProcKind = 32
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
