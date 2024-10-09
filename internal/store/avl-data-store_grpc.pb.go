// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.21.12
// source: avl-data-store.proto

package store

import (
	context "context"
	types "github.com/404minds/avl-receiver/internal/types"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	AvlDataStore_VerifyDevice_FullMethodName       = "/store.AvlDataStore/VerifyDevice"
	AvlDataStore_SaveDeviceStatus_FullMethodName   = "/store.AvlDataStore/SaveDeviceStatus"
	AvlDataStore_SavedeviceResponse_FullMethodName = "/store.AvlDataStore/SavedeviceResponse"
	AvlDataStore_FetchDeviceModel_FullMethodName   = "/store.AvlDataStore/FetchDeviceModel"
)

// AvlDataStoreClient is the client API for AvlDataStore service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AvlDataStoreClient interface {
	VerifyDevice(ctx context.Context, in *VerifyDeviceRequest, opts ...grpc.CallOption) (*VerifyDeviceReply, error)
	SaveDeviceStatus(ctx context.Context, in *types.DeviceStatus, opts ...grpc.CallOption) (*emptypb.Empty, error)
	SavedeviceResponse(ctx context.Context, in *types.DeviceResponse, opts ...grpc.CallOption) (*emptypb.Empty, error)
	FetchDeviceModel(ctx context.Context, in *FetchDeviceModelRequest, opts ...grpc.CallOption) (*FetchDeviceModelResponse, error)
}

type avlDataStoreClient struct {
	cc grpc.ClientConnInterface
}

func NewAvlDataStoreClient(cc grpc.ClientConnInterface) AvlDataStoreClient {
	return &avlDataStoreClient{cc}
}

func (c *avlDataStoreClient) VerifyDevice(ctx context.Context, in *VerifyDeviceRequest, opts ...grpc.CallOption) (*VerifyDeviceReply, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(VerifyDeviceReply)
	err := c.cc.Invoke(ctx, AvlDataStore_VerifyDevice_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *avlDataStoreClient) SaveDeviceStatus(ctx context.Context, in *types.DeviceStatus, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, AvlDataStore_SaveDeviceStatus_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *avlDataStoreClient) SavedeviceResponse(ctx context.Context, in *types.DeviceResponse, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, AvlDataStore_SavedeviceResponse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *avlDataStoreClient) FetchDeviceModel(ctx context.Context, in *FetchDeviceModelRequest, opts ...grpc.CallOption) (*FetchDeviceModelResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(FetchDeviceModelResponse)
	err := c.cc.Invoke(ctx, AvlDataStore_FetchDeviceModel_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AvlDataStoreServer is the server API for AvlDataStore service.
// All implementations must embed UnimplementedAvlDataStoreServer
// for forward compatibility.
type AvlDataStoreServer interface {
	VerifyDevice(context.Context, *VerifyDeviceRequest) (*VerifyDeviceReply, error)
	SaveDeviceStatus(context.Context, *types.DeviceStatus) (*emptypb.Empty, error)
	SavedeviceResponse(context.Context, *types.DeviceResponse) (*emptypb.Empty, error)
	FetchDeviceModel(context.Context, *FetchDeviceModelRequest) (*FetchDeviceModelResponse, error)
	mustEmbedUnimplementedAvlDataStoreServer()
}

// UnimplementedAvlDataStoreServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedAvlDataStoreServer struct{}

func (UnimplementedAvlDataStoreServer) VerifyDevice(context.Context, *VerifyDeviceRequest) (*VerifyDeviceReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method VerifyDevice not implemented")
}
func (UnimplementedAvlDataStoreServer) SaveDeviceStatus(context.Context, *types.DeviceStatus) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SaveDeviceStatus not implemented")
}
func (UnimplementedAvlDataStoreServer) SavedeviceResponse(context.Context, *types.DeviceResponse) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SavedeviceResponse not implemented")
}
func (UnimplementedAvlDataStoreServer) FetchDeviceModel(context.Context, *FetchDeviceModelRequest) (*FetchDeviceModelResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchDeviceModel not implemented")
}
func (UnimplementedAvlDataStoreServer) mustEmbedUnimplementedAvlDataStoreServer() {}
func (UnimplementedAvlDataStoreServer) testEmbeddedByValue()                      {}

// UnsafeAvlDataStoreServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AvlDataStoreServer will
// result in compilation errors.
type UnsafeAvlDataStoreServer interface {
	mustEmbedUnimplementedAvlDataStoreServer()
}

func RegisterAvlDataStoreServer(s grpc.ServiceRegistrar, srv AvlDataStoreServer) {
	// If the following call pancis, it indicates UnimplementedAvlDataStoreServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
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
		FullMethod: AvlDataStore_VerifyDevice_FullMethodName,
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
		FullMethod: AvlDataStore_SaveDeviceStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AvlDataStoreServer).SaveDeviceStatus(ctx, req.(*types.DeviceStatus))
	}
	return interceptor(ctx, in, info, handler)
}

func _AvlDataStore_SavedeviceResponse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(types.DeviceResponse)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AvlDataStoreServer).SavedeviceResponse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: AvlDataStore_SavedeviceResponse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AvlDataStoreServer).SavedeviceResponse(ctx, req.(*types.DeviceResponse))
	}
	return interceptor(ctx, in, info, handler)
}

func _AvlDataStore_FetchDeviceModel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FetchDeviceModelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AvlDataStoreServer).FetchDeviceModel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: AvlDataStore_FetchDeviceModel_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AvlDataStoreServer).FetchDeviceModel(ctx, req.(*FetchDeviceModelRequest))
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
		{
			MethodName: "SavedeviceResponse",
			Handler:    _AvlDataStore_SavedeviceResponse_Handler,
		},
		{
			MethodName: "FetchDeviceModel",
			Handler:    _AvlDataStore_FetchDeviceModel_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "avl-data-store.proto",
}
