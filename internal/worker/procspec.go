package worker

import (
	"fmt"

	"github.com/djosix/med/internal/helper"
	pb "github.com/djosix/med/internal/protobuf"
)

func SendProcSpec(ctx *ProcRunCtx, spec any) {
	ctx.OutputPacket(&pb.Packet{
		Kind: pb.PacketKind_PacketKindInfo,
		Data: helper.MustEncode(spec),
	})
}

func RecvProcSpec[T any](ctx *ProcRunCtx) (*T, error) {
	pkt := ctx.InputPacket()
	if pkt == nil {
		return nil, fmt.Errorf("input channel closed")
	}
	if pkt.Kind != pb.PacketKind_PacketKindInfo {
		return nil, fmt.Errorf("not an info packet")
	}
	spec, err := helper.DecodeAs[T](pkt.Data)
	if err != nil {
		return nil, err
	}
	return &spec, nil
}
