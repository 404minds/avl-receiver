package store

import (
	"context"
	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

type CustomAvlDataStoreClient struct {
	cc grpc.ClientConnInterface
}

func (c CustomAvlDataStoreClient) VerifyDevice(ctx context.Context, in *VerifyDeviceRequest, opts ...grpc.CallOption) (*VerifyDeviceReply, error) {
	out := new(VerifyDeviceReply)
	err := c.cc.Invoke(ctx, "/AVLService/VerifyDevice", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c CustomAvlDataStoreClient) SaveDeviceStatus(ctx context.Context, in *types.DeviceStatus, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)

	start := time.Now()
	err := c.cc.Invoke(ctx, "/AVLService/InsertAVL", in, out, opts...)

	duration := time.Since(start)
	logger.Sugar().Info("time taken to complete go routine ", duration.Seconds())
	if err != nil {
		return nil, err
	}
	return out, nil
}

func NewCustomAvlDataStoreClient(cc grpc.ClientConnInterface) *CustomAvlDataStoreClient {
	return &CustomAvlDataStoreClient{cc: cc}
}
