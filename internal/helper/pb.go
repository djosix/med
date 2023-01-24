package helper

import pb "github.com/djosix/med/internal/protobuf"

func NewDataPkt(data []byte) *pb.Packet {
	return &pb.Packet{
		Kind: pb.PacketKind_PacketKindData,
		Data: data,
	}
}

func NewCtrlPkt(data []byte) *pb.Packet {
	return &pb.Packet{
		Kind: pb.PacketKind_PacketKindCtrl,
		Data: data,
	}
}

func NewErrorPkt(data []byte) *pb.Packet {
	return &pb.Packet{
		Kind: pb.PacketKind_PacketKindError,
		Data: data,
	}
}
