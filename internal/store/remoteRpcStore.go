package store

import (
	"context"
	"github.com/404minds/avl-receiver/internal/types"
	"go.uber.org/zap"
)

type RemoteRpcStore struct {
	ProcessChan       chan *types.DeviceStatus // TODO: change to a more specific type
	ResponseChan      chan *types.DeviceResponse
	CloseChan         chan bool
	CloseResponseChan chan bool
	RemoteStoreClient CustomAvlDataStoreClient
	DeviceIdentifier  string
}

func (s *RemoteRpcStore) GetProcessChan() chan *types.DeviceStatus {
	return s.ProcessChan
}

func (s *RemoteRpcStore) GetResponseChan() chan *types.DeviceResponse {
	return s.ResponseChan
}

func (s *RemoteRpcStore) GetCloseChan() chan bool {
	return s.CloseChan
}

func (s *RemoteRpcStore) GetCloseResponseChan() chan bool {
	return s.CloseResponseChan
}

func (s *RemoteRpcStore) Process(ctx context.Context) {
	for {
		select {
		case deviceStatus := <-s.ProcessChan:
			logger.Sugar().Infoln(deviceStatus.String())
			ctx := context.Background()
			_, err := s.RemoteStoreClient.SaveDeviceStatus(ctx, deviceStatus)
			if err != nil {
				logger.Error("failed to save device status", zap.String("imei", deviceStatus.Imei), zap.Error(err))
			}
		case <-s.CloseChan:
			logger.Sugar().Info("async remote rpc store shutting down for device")
			return
		case <-ctx.Done():
			logger.Sugar().Info("context canceled, shutting down process goroutine")
			return

		}
	}
}

func (s *RemoteRpcStore) Response(ctx context.Context) {
	for {
		select {
		case deviceResponse := <-s.ResponseChan:
			logger.Sugar().Info(deviceResponse.String())

			_, err := s.RemoteStoreClient.SaveDeviceResponse(ctx, deviceResponse)
			if err != nil {
				logger.Error("failed to save device status", zap.String("imei", deviceResponse.Imei), zap.Error(err))
			}

		case <-s.CloseResponseChan:
			logger.Sugar().Info("async remote rpc store shutting down for device")
			return
		case <-ctx.Done():
			logger.Sugar().Info("context canceled, shutting down response goroutine")
			return

		}

	}
}
