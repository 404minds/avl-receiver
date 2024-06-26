// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: avl-data-store.proto

package store

import (
	context "context"
	types "github.com/404minds/avl-receiver/internal/types"
	empty "github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// AvlDataStoreClient is the client API for AvlDataStore service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AvlDataStoreClient interface {
	VerifyDevice(ctx context.Context, in *VerifyDeviceRequest, opts ...grpc.CallOption) (*VerifyDeviceReply, error)
	SaveDeviceStatus(ctx context.Context, in *types.DeviceStatus, opts ...grpc.CallOption) (*empty.Empty, error)
}

type avlDataStoreClient struct {
	cc grpc.ClientConnInterface
}

func NewAvlDataStoreClient(cc grpc.ClientConnInterface) AvlDataStoreClient {
	return &avlDataStoreClient{cc}
}

func (c *avlDataStoreClient) VerifyDevice(ctx context.Context, in *VerifyDeviceRequest, opts ...grpc.CallOption) (*VerifyDeviceReply, error) {
	out := new(VerifyDeviceReply)
	err := c.cc.Invoke(ctx, "/store.AvlDataStore/VerifyDevice", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *avlDataStoreClient) SaveDeviceStatus(ctx context.Context, in *types.DeviceStatus, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/store.AvlDataStore/SaveDeviceStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AvlDataStoreServer is the server API for AvlDataStore service.
// All implementations must embed UnimplementedAvlDataStoreServer
// for forward compatibility
type AvlDataStoreServer interface {
	VerifyDevice(context.Context, *VerifyDeviceRequest) (*VerifyDeviceReply, error)
	SaveDeviceStatus(context.Context, *types.DeviceStatus) (*empty.Empty, error)
	mustEmbedUnimplementedAvlDataStoreServer()
}

// UnimplementedAvlDataStoreServer must be embedded to have forward compatible implementations.
type UnimplementedAvlDataStoreServer struct {
}

func (UnimplementedAvlDataStoreServer) VerifyDevice(context.Context, *VerifyDeviceRequest) (*VerifyDeviceReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method VerifyDevice not implemented")
}
func (UnimplementedAvlDataStoreServer) SaveDeviceStatus(context.Context, *types.DeviceStatus) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SaveDeviceStatus not implemented")
}
func (UnimplementedAvlDataStoreServer) mustEmbedUnimplementedAvlDataStoreServer() {}

// UnsafeAvlDataStoreServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AvlDataStoreServer will
// result in compilation errors.
type UnsafeAvlDataStoreServer interface {
	mustEmbedUnimplementedAvlDataStoreServer()
}

func RegisterAvlDataStoreServer(s grpc.ServiceRegistrar, srv AvlDataStoreServer) {
	s.RegisterService(&AvlDataStore_ServiceDesc, srv)
}

func _AvlDataStore_VerifyDevice_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VerifyDeviceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AvlDataStoreServer).VerifyDevice(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/store.AvlDataStore/VerifyDevice",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AvlDataStoreServer).VerifyDevice(ctx, req.(*VerifyDeviceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _AvlDataStore_SaveDeviceStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(types.DeviceStatus)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AvlDataStoreServer).SaveDeviceStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/store.AvlDataStore/SaveDeviceStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AvlDataStoreServer).SaveDeviceStatus(ctx, req.(*types.DeviceStatus))
	}
	return interceptor(ctx, in, info, handler)
}

// AvlDataStore_ServiceDesc is the grpc.ServiceDesc for AvlDataStore service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AvlDataStore_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "store.AvlDataStore",
	HandlerType: (*AvlDataStoreServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "VerifyDevice",
			Handler:    _AvlDataStore_VerifyDevice_Handler,
		},
		{
			MethodName: "SaveDeviceStatus",
			Handler:    _AvlDataStore_SaveDeviceStatus_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "avl-data-store.proto",
}
