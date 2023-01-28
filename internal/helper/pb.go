package helper

import pb "github.com/djosix/med/internal/protobuf"

func NewDataPacket(data []byte) *pb.Packet {
	return &pb.Packet{
		Kind: pb.PacketKind_PacketKindData,
		Data: data,
	}
}

func NewCtrlPacket(data []byte) *pb.Packet {
	return &pb.Packet{
		Kind: pb.PacketKind_PacketKindCtrl,
		Data: data,
	}
}

func NewErrorPacket(s string) *pb.Packet {
	return &pb.Packet{
		Kind: pb.PacketKind_PacketKindError,
		Data: []byte(s),
	}
}
