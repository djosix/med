// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.12
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

type KexReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Nonce         []byte `protobuf:"bytes,1,opt,name=nonce,proto3" json:"nonce,omitempty"`
	PublicKeyHash []byte `protobuf:"bytes,2,opt,name=public_key_hash,json=publicKeyHash,proto3" json:"public_key_hash,omitempty"`
	Signature     []byte `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"`
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

	Nonce        []byte `protobuf:"bytes,1,opt,name=nonce,proto3" json:"nonce,omitempty"`
	KexPublicKey []byte `protobuf:"bytes,2,opt,name=kex_public_key,json=kexPublicKey,proto3" json:"kex_public_key,omitempty"`
	Signature    []byte `protobuf:"bytes,3,opt,name=signature,proto3" json:"signature,omitempty"`
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

var File_med_proto protoreflect.FileDescriptor

var file_med_proto_rawDesc = []byte{
	0x0a, 0x09, 0x6d, 0x65, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x64, 0x0a, 0x06, 0x4b, 0x65, 0x78, 0x52, 0x65, 0x71, 0x12, 0x14, 0x0a, 0x05,
	0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x05, 0x6e, 0x6f, 0x6e,
	0x63, 0x65, 0x12, 0x26, 0x0a, 0x0f, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x6b, 0x65, 0x79,
	0x5f, 0x68, 0x61, 0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0d, 0x70, 0x75, 0x62,
	0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x48, 0x61, 0x73, 0x68, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x69,
	0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x73,
	0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x22, 0x63, 0x0a, 0x07, 0x4b, 0x65, 0x78, 0x52,
	0x65, 0x73, 0x70, 0x12, 0x14, 0x0a, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x12, 0x24, 0x0a, 0x0e, 0x6b, 0x65, 0x78,
	0x5f, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x6b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x0c, 0x6b, 0x65, 0x78, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x12,
	0x1c, 0x0a, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0c, 0x52, 0x09, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x74, 0x75, 0x72, 0x65, 0x42, 0x15, 0x5a,
	0x13, 0x2e, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
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

var file_med_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_med_proto_goTypes = []interface{}{
	(*KexReq)(nil),  // 0: proto.KexReq
	(*KexResp)(nil), // 1: proto.KexResp
}
var file_med_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
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
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_med_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_med_proto_goTypes,
		DependencyIndexes: file_med_proto_depIdxs,
		MessageInfos:      file_med_proto_msgTypes,
	}.Build()
	File_med_proto = out.File
	file_med_proto_rawDesc = nil
	file_med_proto_goTypes = nil
	file_med_proto_depIdxs = nil
}
