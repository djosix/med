package worker

import (
	"fmt"

	"github.com/djosix/med/internal/helper"
	pb "github.com/djosix/med/internal/protobuf"
)

func SendProcSpec(ctx *ProcRunCtx, spec any) {
	ctx.PktOutCh <- &pb.Packet{
		Kind: pb.PacketKind_PacketKindInfo,
		Data: helper.MustEncode(spec),
	}
}

func RecvProcSpec[T any](ctx *ProcRunCtx) (*T, error) {
	select {
	case pkt := <-ctx.PktInCh:
		if pkt == nil {
			return nil, fmt.Errorf("input channel closed")
		}
		spec, err := helper.DecodeAs[T](pkt.Data)
		if err != nil {
			return nil, err
		}
		return &spec, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled")
	}
}
