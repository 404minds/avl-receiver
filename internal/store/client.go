package store

import (
	"context"

	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type CustomAvlDataStoreClient struct {
	cc          grpc.ClientConnInterface
	serviceName string
}

func (c CustomAvlDataStoreClient) VerifyDevice(ctx context.Context, in *VerifyDeviceRequest, opts ...grpc.CallOption) (*VerifyDeviceReply, error) {
	out := new(VerifyDeviceReply)
	err := c.cc.Invoke(ctx, c.serviceName+"/VerifyDevice", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c CustomAvlDataStoreClient) SaveDeviceStatus(ctx context.Context, in *types.DeviceStatus, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, c.serviceName+"/InsertAVL", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c CustomAvlDataStoreClient) SaveDeviceResponse(ctx context.Context, in *types.DeviceResponse, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, c.serviceName+"/InsertDeviceResponse", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c CustomAvlDataStoreClient) FetchDeviceModel(ctx context.Context, in *types.FetchDeviceModelRequest, opts ...grpc.CallOption) (*FetchDeviceModelResponse, error) {
	out := new(FetchDeviceModelResponse)
	err := c.cc.Invoke(ctx, c.serviceName+"/FetchDeviceModel", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func NewCustomAvlDataStoreClient(cc grpc.ClientConnInterface, serviceName string) *CustomAvlDataStoreClient {
	return &CustomAvlDataStoreClient{
		cc:          cc,
		serviceName: serviceName,
	}
}
