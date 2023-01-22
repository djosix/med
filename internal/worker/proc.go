package worker

import (
	"context"

	pb "github.com/djosix/med/internal/protobuf"
)

type Proc interface {
	Run(ctx ProcRunCtx)
}

type ProcRunCtx struct {
	context.Context
	Cancel   context.CancelFunc
	Loop     Loop
	MsgOutCh chan<- *pb.MedMsg
	MsgInCh  <-chan *pb.MedMsg
}
