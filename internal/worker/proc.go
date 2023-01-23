package worker

import (
	"context"

	pb "github.com/djosix/med/internal/protobuf"
)

type ProcKind byte

const (
	ProcKind_Exec     ProcKind = 11
	ProcKind_Term     ProcKind = 12
	ProcKind_LocalPF  ProcKind = 21
	ProcKind_RemotePF ProcKind = 22
	ProcKind_Upload   ProcKind = 31
	ProcKind_Download ProcKind = 32
)

type Proc interface {
	Run(ctx ProcRunCtx)
}

type ProcRunCtx struct {
	context.Context
	Cancel   context.CancelFunc
	Loop     Loop
	PktOutCh chan<- *pb.Packet
	PktInCh  <-chan *pb.Packet
	ProcID   uint32
}
