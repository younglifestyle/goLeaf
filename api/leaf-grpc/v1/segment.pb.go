// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.17.1
// source: leaf-grpc/v1/segment.proto

package v1

import (
	_ "google.golang.org/genproto/googleapis/api/annotations"
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

// 申请ID的BIZ Tag
type IDRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Tag string `protobuf:"bytes,1,opt,name=tag,proto3" json:"tag,omitempty"`
}

func (x *IDRequest) Reset() {
	*x = IDRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_leaf_grpc_v1_segment_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IDRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IDRequest) ProtoMessage() {}

func (x *IDRequest) ProtoReflect() protoreflect.Message {
	mi := &file_leaf_grpc_v1_segment_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IDRequest.ProtoReflect.Descriptor instead.
func (*IDRequest) Descriptor() ([]byte, []int) {
	return file_leaf_grpc_v1_segment_proto_rawDescGZIP(), []int{0}
}

func (x *IDRequest) GetTag() string {
	if x != nil {
		return x.Tag
	}
	return ""
}

// 申请到的ID
type IDReply struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *IDReply) Reset() {
	*x = IDReply{}
	if protoimpl.UnsafeEnabled {
		mi := &file_leaf_grpc_v1_segment_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IDReply) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IDReply) ProtoMessage() {}

func (x *IDReply) ProtoReflect() protoreflect.Message {
	mi := &file_leaf_grpc_v1_segment_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IDReply.ProtoReflect.Descriptor instead.
func (*IDReply) Descriptor() ([]byte, []int) {
	return file_leaf_grpc_v1_segment_proto_rawDescGZIP(), []int{1}
}

func (x *IDReply) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

var File_leaf_grpc_v1_segment_proto protoreflect.FileDescriptor

var file_leaf_grpc_v1_segment_proto_rawDesc = []byte{
	0x0a, 0x1a, 0x6c, 0x65, 0x61, 0x66, 0x2d, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x76, 0x31, 0x2f, 0x73,
	0x65, 0x67, 0x6d, 0x65, 0x6e, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0b, 0x6c, 0x65,
	0x61, 0x66, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x76, 0x31, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x1d, 0x0a, 0x09, 0x49, 0x44, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x74, 0x61, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x03, 0x74, 0x61, 0x67, 0x22, 0x19, 0x0a, 0x07, 0x49, 0x44, 0x52, 0x65, 0x70, 0x6c,
	0x79, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x69,
	0x64, 0x32, 0x64, 0x0a, 0x04, 0x4c, 0x65, 0x61, 0x66, 0x12, 0x5c, 0x0a, 0x0c, 0x47, 0x65, 0x6e,
	0x53, 0x65, 0x67, 0x6d, 0x65, 0x6e, 0x74, 0x49, 0x64, 0x12, 0x16, 0x2e, 0x6c, 0x65, 0x61, 0x66,
	0x67, 0x72, 0x70, 0x63, 0x2e, 0x76, 0x31, 0x2e, 0x49, 0x44, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x14, 0x2e, 0x6c, 0x65, 0x61, 0x66, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x76, 0x31, 0x2e,
	0x49, 0x44, 0x52, 0x65, 0x70, 0x6c, 0x79, 0x22, 0x1e, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x18, 0x12,
	0x16, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x73, 0x65, 0x67, 0x6d, 0x65, 0x6e, 0x74, 0x2f, 0x67, 0x65,
	0x74, 0x2f, 0x7b, 0x74, 0x61, 0x67, 0x7d, 0x42, 0x20, 0x5a, 0x1e, 0x73, 0x65, 0x71, 0x2d, 0x73,
	0x65, 0x72, 0x76, 0x65, 0x72, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x6c, 0x65, 0x61, 0x66, 0x2d, 0x67,
	0x72, 0x70, 0x63, 0x2f, 0x76, 0x31, 0x3b, 0x76, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x33,
}

var (
	file_leaf_grpc_v1_segment_proto_rawDescOnce sync.Once
	file_leaf_grpc_v1_segment_proto_rawDescData = file_leaf_grpc_v1_segment_proto_rawDesc
)

func file_leaf_grpc_v1_segment_proto_rawDescGZIP() []byte {
	file_leaf_grpc_v1_segment_proto_rawDescOnce.Do(func() {
		file_leaf_grpc_v1_segment_proto_rawDescData = protoimpl.X.CompressGZIP(file_leaf_grpc_v1_segment_proto_rawDescData)
	})
	return file_leaf_grpc_v1_segment_proto_rawDescData
}

var file_leaf_grpc_v1_segment_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_leaf_grpc_v1_segment_proto_goTypes = []interface{}{
	(*IDRequest)(nil), // 0: leafgrpc.v1.IDRequest
	(*IDReply)(nil),   // 1: leafgrpc.v1.IDReply
}
var file_leaf_grpc_v1_segment_proto_depIdxs = []int32{
	0, // 0: leafgrpc.v1.Leaf.GenSegmentId:input_type -> leafgrpc.v1.IDRequest
	1, // 1: leafgrpc.v1.Leaf.GenSegmentId:output_type -> leafgrpc.v1.IDReply
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_leaf_grpc_v1_segment_proto_init() }
func file_leaf_grpc_v1_segment_proto_init() {
	if File_leaf_grpc_v1_segment_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_leaf_grpc_v1_segment_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IDRequest); i {
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
		file_leaf_grpc_v1_segment_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IDReply); i {
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
			RawDescriptor: file_leaf_grpc_v1_segment_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_leaf_grpc_v1_segment_proto_goTypes,
		DependencyIndexes: file_leaf_grpc_v1_segment_proto_depIdxs,
		MessageInfos:      file_leaf_grpc_v1_segment_proto_msgTypes,
	}.Build()
	File_leaf_grpc_v1_segment_proto = out.File
	file_leaf_grpc_v1_segment_proto_rawDesc = nil
	file_leaf_grpc_v1_segment_proto_goTypes = nil
	file_leaf_grpc_v1_segment_proto_depIdxs = nil
}