// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.12.4
// source: med.proto

package protobuf

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type MedMsgType int32

const (
	MedMsgType_MedMsgTypeNone    MedMsgType = 0
	MedMsgType_MedMsgTypeControl MedMsgType = 1
	MedMsgType_MedMsgTypeData    MedMsgType = 2
	MedMsgType_MedMsgTypeError   MedMsgType = 3
)

// Enum value maps for MedMsgType.
var (
	MedMsgType_name = map[int32]string{
		0: "MedMsgTypeNone",
		1: "MedMsgTypeControl",
		2: "MedMsgTypeData",
		3: "MedMsgTypeError",
	}
	MedMsgType_value = map[string]int32{
		"MedMsgTypeNone":    0,
		"MedMsgTypeControl": 1,
		"MedMsgTypeData":    2,
		"MedMsgTypeError":   3,
	}
)

func (x MedMsgType) Enum() *MedMsgType {
	p := new(MedMsgType)
	*p = x
	return p
}

func (x MedMsgType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (MedMsgType) Descriptor() protoreflect.EnumDescriptor {
	return file_med_proto_enumTypes[0].Descriptor()
}

func (MedMsgType) Type() protoreflect.EnumType {
	return &file_med_proto_enumTypes[0]
}

func (x MedMsgType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use MedMsgType.Descriptor instead.
func (MedMsgType) EnumDescriptor() ([]byte, []int) {
	return file_med_proto_rawDescGZIP(), []int{0}
}

type KexReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Nonce         []byte `protobuf:"bytes,1,opt,name=Nonce,proto3" json:"Nonce,omitempty"`
	PublicKeyHash []byte `protobuf:"bytes,2,opt,name=PublicKeyHash,proto3" json:"PublicKeyHash,omitempty"`
	Signature     []byte `protobuf:"bytes,3,opt,name=Signature,proto3" json:"Signature,omitempty"`
}

func (x *KexReq) Reset() {
	*x = KexReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_med_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *KexReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*KexReq) ProtoMessage() {}

func (x *KexReq) ProtoReflect() protoreflect.Message {
	mi := &file_med_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use KexReq.ProtoReflect.Descriptor instead.
func (*KexReq) Descriptor() ([]byte, []int) {
	return file_med_proto_rawDescGZIP(), []int{0}
}

func (x *KexReq) GetNonce() []byte {
	if x != nil {
		return x.Nonce
	}
	return nil
}

func (x *KexReq) GetPublicKeyHash() []byte {
	if x != nil {
		return x.PublicKeyHash
	}
	return nil
}

func (x *KexReq) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

type KexResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Nonce        []byte `protobuf:"bytes,1,opt,name=Nonce,proto3" json:"Nonce,omitempty"`
	KexPublicKey []byte `protobuf:"bytes,2,opt,name=KexPublicKey,proto3" json:"KexPublicKey,omitempty"`
	Signature    []byte `protobuf:"bytes,3,opt,name=Signature,proto3" json:"Signature,omitempty"`
}

func (x *KexResp) Reset() {
	*x = KexResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_med_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *KexResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*KexResp) ProtoMessage() {}

func (x *KexResp) ProtoReflect() protoreflect.Message {
	mi := &file_med_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use KexResp.ProtoReflect.Descriptor instead.
func (*KexResp) Descriptor() ([]byte, []int) {
	return file_med_proto_rawDescGZIP(), []int{1}
}

func (x *KexResp) GetNonce() []byte {
	if x != nil {
		return x.Nonce
	}
	return nil
}

func (x *KexResp) GetKexPublicKey() []byte {
	if x != nil {
		return x.KexPublicKey
	}
	return nil
}

func (x *KexResp) GetSignature() []byte {
	if x != nil {
		return x.Signature
	}
	return nil
}

type MedPkt struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	TargetID uint32  `protobuf:"varint,1,opt,name=TargetID,proto3" json:"TargetID,omitempty"`
	SourceID uint32  `protobuf:"varint,2,opt,name=SourceID,proto3" json:"SourceID,omitempty"`
	Message  *MedMsg `protobuf:"bytes,3,opt,name=Message,proto3" json:"Message,omitempty"`
}

func (x *MedPkt) Reset() {
	*x = MedPkt{}
	if protoimpl.UnsafeEnabled {
		mi := &file_med_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MedPkt) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MedPkt) ProtoMessage() {}

func (x *MedPkt) ProtoReflect() protoreflect.Message {
	mi := &file_med_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MedPkt.ProtoReflect.Descriptor instead.
func (*MedPkt) Descriptor() ([]byte, []int) {
	return file_med_proto_rawDescGZIP(), []int{2}
}

func (x *MedPkt) GetTargetID() uint32 {
	if x != nil {
		return x.TargetID
	}
	return 0
}

func (x *MedPkt) GetSourceID() uint32 {
	if x != nil {
		return x.SourceID
	}
	return 0
}

func (x *MedPkt) GetMessage() *MedMsg {
	if x != nil {
		return x.Message
	}
	return nil
}

type MedMsg struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Type MedMsgType `protobuf:"varint,1,opt,name=Type,proto3,enum=proto.MedMsgType" json:"Type,omitempty"`
	Data []byte     `protobuf:"bytes,2,opt,name=Data,proto3" json:"Data,omitempty"`
}

func (x *MedMsg) Reset() {
	*x = MedMsg{}
	if protoimpl.UnsafeEnabled {
		mi := &file_med_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MedMsg) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MedMsg) ProtoMessage() {}

func (x *MedMsg) ProtoReflect() protoreflect.Message {
	mi := &file_med_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MedMsg.ProtoReflect.Descriptor instead.
func (*MedMsg) Descriptor() ([]byte, []int) {
	return file_med_proto_rawDescGZIP(), []int{3}
}

func (x *MedMsg) GetType() MedMsgType {
	if x != nil {
		return x.Type
	}
	return MedMsgType_MedMsgTypeNone
}

func (x *MedMsg) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

var File_med_proto protoreflect.FileDescriptor

var file_med_proto_rawDesc = []byte{
	0x0a, 0x09, 0x6d, 0x65, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x62, 0x0a, 0x06, 0x4b, 0x65, 0x78, 0x52, 0x65, 0x71, 0x12, 0x14, 0x0a, 0x05,
	0x4e, 0x6f, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x4e, 0x6f, 0x6e,
	0x63, 0x65, 0x12, 0x24, 0x0a, 0x0d, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x48,
	0x61, 0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0d, 0x50, 0x75, 0x62, 0x6c, 0x69,
	0x63, 0x4b, 0x65, 0x79, 0x48, 0x61, 0x73, 0x68, 0x12, 0x1c, 0x0a, 0x09, 0x53, 0x69, 0x67, 0x6e,
	0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x53, 0x69, 0x67,
	0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x22, 0x61, 0x0a, 0x07, 0x4b, 0x65, 0x78, 0x52, 0x65, 0x73,
	0x70, 0x12, 0x14, 0x0a, 0x05, 0x4e, 0x6f, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x05, 0x4e, 0x6f, 0x6e, 0x63, 0x65, 0x12, 0x22, 0x0a, 0x0c, 0x4b, 0x65, 0x78, 0x50, 0x75,
	0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0c, 0x4b,
	0x65, 0x78, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x12, 0x1c, 0x0a, 0x09, 0x53,
	0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09,
	0x53, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x22, 0x69, 0x0a, 0x06, 0x4d, 0x65, 0x64,
	0x50, 0x6b, 0x74, 0x12, 0x1a, 0x0a, 0x08, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x49, 0x44, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x54, 0x61, 0x72, 0x67, 0x65, 0x74, 0x49, 0x44, 0x12,
	0x1a, 0x0a, 0x08, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x44, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0d, 0x52, 0x08, 0x53, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x49, 0x44, 0x12, 0x27, 0x0a, 0x07, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4d, 0x65, 0x64, 0x4d, 0x73, 0x67, 0x52, 0x07, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x22, 0x43, 0x0a, 0x06, 0x4d, 0x65, 0x64, 0x4d, 0x73, 0x67, 0x12, 0x25,
	0x0a, 0x04, 0x54, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x11, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x4d, 0x65, 0x64, 0x4d, 0x73, 0x67, 0x54, 0x79, 0x70, 0x65, 0x52,
	0x04, 0x54, 0x79, 0x70, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x44, 0x61, 0x74, 0x61, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x04, 0x44, 0x61, 0x74, 0x61, 0x2a, 0x60, 0x0a, 0x0a, 0x4d, 0x65, 0x64,
	0x4d, 0x73, 0x67, 0x54, 0x79, 0x70, 0x65, 0x12, 0x12, 0x0a, 0x0e, 0x4d, 0x65, 0x64, 0x4d, 0x73,
	0x67, 0x54, 0x79, 0x70, 0x65, 0x4e, 0x6f, 0x6e, 0x65, 0x10, 0x00, 0x12, 0x15, 0x0a, 0x11, 0x4d,
	0x65, 0x64, 0x4d, 0x73, 0x67, 0x54, 0x79, 0x70, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x72, 0x6f, 0x6c,
	0x10, 0x01, 0x12, 0x12, 0x0a, 0x0e, 0x4d, 0x65, 0x64, 0x4d, 0x73, 0x67, 0x54, 0x79, 0x70, 0x65,
	0x44, 0x61, 0x74, 0x61, 0x10, 0x02, 0x12, 0x13, 0x0a, 0x0f, 0x4d, 0x65, 0x64, 0x4d, 0x73, 0x67,
	0x54, 0x79, 0x70, 0x65, 0x45, 0x72, 0x72, 0x6f, 0x72, 0x10, 0x03, 0x42, 0x15, 0x5a, 0x13, 0x2e,
	0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_med_proto_rawDescOnce sync.Once
	file_med_proto_rawDescData = file_med_proto_rawDesc
)

func file_med_proto_rawDescGZIP() []byte {
	file_med_proto_rawDescOnce.Do(func() {
		file_med_proto_rawDescData = protoimpl.X.CompressGZIP(file_med_proto_rawDescData)
	})
	return file_med_proto_rawDescData
}

var file_med_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_med_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_med_proto_goTypes = []interface{}{
	(MedMsgType)(0), // 0: proto.MedMsgType
	(*KexReq)(nil),  // 1: proto.KexReq
	(*KexResp)(nil), // 2: proto.KexResp
	(*MedPkt)(nil),  // 3: proto.MedPkt
	(*MedMsg)(nil),  // 4: proto.MedMsg
}
var file_med_proto_depIdxs = []int32{
	4, // 0: proto.MedPkt.Message:type_name -> proto.MedMsg
	0, // 1: proto.MedMsg.Type:type_name -> proto.MedMsgType
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_med_proto_init() }
func file_med_proto_init() {
	if File_med_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_med_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*KexReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_med_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*KexResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_med_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MedPkt); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_med_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MedMsg); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_med_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_med_proto_goTypes,
		DependencyIndexes: file_med_proto_depIdxs,
		EnumInfos:         file_med_proto_enumTypes,
		MessageInfos:      file_med_proto_msgTypes,
	}.Build()
	File_med_proto = out.File
	file_med_proto_rawDesc = nil
	file_med_proto_goTypes = nil
	file_med_proto_depIdxs = nil
}
