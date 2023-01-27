package protobuf

type PacketCtrlKind byte

const (
	PacketCtrl_None PacketCtrlKind = 0x00
	PacketCtrl_Exit PacketCtrlKind = 0x0e
)

func GetPacketCtrlKind(pkt *Packet) PacketCtrlKind {
	if pkt.Kind != PacketKind_PacketKindCtrl {
		return PacketCtrl_None
	} else if len(pkt.Data) != 1 {
		return PacketCtrl_None
	}
	return PacketCtrlKind(pkt.Data[0])
}

func IsPacketWithCtrlKind(pkt *Packet, kind PacketCtrlKind) bool {
	return GetPacketCtrlKind(pkt) == kind
}

func NewCtrlPacket(targetID uint32, kind PacketCtrlKind) *Packet {
	return &Packet{
		TargetID: targetID,
		Kind:     PacketKind_PacketKindCtrl,
		Data:     []byte{byte(kind)},
	}
}
