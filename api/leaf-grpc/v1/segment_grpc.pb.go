// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.17.1
// source: leaf-grpc/v1/segment.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// LeafClient is the client API for Leaf service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type LeafClient interface {
	// 号段模式
	GenSegmentId(ctx context.Context, in *IDRequest, opts ...grpc.CallOption) (*IDReply, error)
}

type leafClient struct {
	cc grpc.ClientConnInterface
}

func NewLeafClient(cc grpc.ClientConnInterface) LeafClient {
	return &leafClient{cc}
}

func (c *leafClient) GenSegmentId(ctx context.Context, in *IDRequest, opts ...grpc.CallOption) (*IDReply, error) {
	out := new(IDReply)
	err := c.cc.Invoke(ctx, "/leafgrpc.v1.Leaf/GenSegmentId", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LeafServer is the server API for Leaf service.
// All implementations must embed UnimplementedLeafServer
// for forward compatibility
type LeafServer interface {
	// 号段模式
	GenSegmentId(context.Context, *IDRequest) (*IDReply, error)
	mustEmbedUnimplementedLeafServer()
}

// UnimplementedLeafServer must be embedded to have forward compatible implementations.
type UnimplementedLeafServer struct {
}

func (UnimplementedLeafServer) GenSegmentId(context.Context, *IDRequest) (*IDReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenSegmentId not implemented")
}
func (UnimplementedLeafServer) mustEmbedUnimplementedLeafServer() {}

// UnsafeLeafServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to LeafServer will
// result in compilation errors.
type UnsafeLeafServer interface {
	mustEmbedUnimplementedLeafServer()
}

func RegisterLeafServer(s grpc.ServiceRegistrar, srv LeafServer) {
	s.RegisterService(&Leaf_ServiceDesc, srv)
}

func _Leaf_GenSegmentId_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(IDRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LeafServer).GenSegmentId(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/leafgrpc.v1.Leaf/GenSegmentId",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LeafServer).GenSegmentId(ctx, req.(*IDRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Leaf_ServiceDesc is the grpc.ServiceDesc for Leaf service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Leaf_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "leafgrpc.v1.Leaf",
	HandlerType: (*LeafServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GenSegmentId",
			Handler:    _Leaf_GenSegmentId_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "leaf-grpc/v1/segment.proto",
}
