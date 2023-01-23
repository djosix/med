package worker

import (
	"github.com/djosix/med/internal/helper"
	"github.com/djosix/med/internal/logger"
	pb "github.com/djosix/med/internal/protobuf"
)

////////////////////////////////////////////////////////////////////////////////////////////////

type ProcMainMsgKind byte

const (
	ProcMainMsgKind_Start      ProcMainMsgKind = 0
	ProcMainMsgKind_StartResp  ProcMainMsgKind = 1
	ProcMainMsgKind_Remove     ProcMainMsgKind = 2
	ProcMainMsgKind_RemoveResp ProcMainMsgKind = 3
)

type ProcMainMsg struct {
	Kind ProcMainMsgKind
	Data []byte
}

type ProcMainStart struct {
	ProcKind ProcKind
	SpecData []byte
}

type ProcMainRemove struct {
	ProcID uint32
}

type ProcMainProcInfo struct {
	ProcID   uint32
	ProcKind ProcKind
	IsAlive  bool
	Err      error
}

////////////////////////////////////////////////////////////////////////////////////////////////

type ClientMainProc struct{}

func NewClientMainProc() *ClientMainProc {
	return &ClientMainProc{}
}

func (p *ClientMainProc) Run(ctx ProcRunCtx) {
	logger := logger.NewLogger("ClientMainProc")
	logger.Info("start")
	defer logger.Info("done")

	start := func(procKind ProcKind, spec any) {
		ctx.PktOutCh <- &pb.Packet{
			Kind: pb.PacketKind_PacketKindInfo,
			Data: helper.MustEncode(&ProcMainMsg{
				Kind: ProcMainMsgKind_Start,
				Data: helper.MustEncode(&ProcMainStart{
					ProcKind: procKind,
					SpecData: helper.MustEncode(spec),
				}),
			}),
		}
	}
	_ = start

	remove := func(procID uint32) {
		ctx.PktOutCh <- &pb.Packet{
			Kind: pb.PacketKind_PacketKindInfo,
			Data: helper.MustEncode(&ProcMainMsg{
				Kind: ProcMainMsgKind_Start,
				Data: helper.MustEncode(&ProcMainRemove{
					ProcID: procID,
				}),
			}),
		}
	}
	_ = remove

	handlePacket := func(pkt *pb.Packet) {
		switch pkt.Kind {
		case pb.PacketKind_PacketKindInfo:
		case pb.PacketKind_PacketKindCtrl:
		case pb.PacketKind_PacketKindData:
		case pb.PacketKind_PacketKindError:
		default:
		}
	}

	for {
		var pkt *pb.Packet
		select {
		case pkt = <-ctx.PktInCh:
		case <-ctx.Done():
			break
		}

		logger.Print("pkt:", pkt)
		handlePacket(pkt)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////

type ServerMainProc struct{}

func NewServerMainProc() *ServerMainProc {
	return &ServerMainProc{}
}

func (p *ServerMainProc) Run(ctx ProcRunCtx) {

}
