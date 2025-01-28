package store

import (
	"context"
	"time"

	"github.com/404minds/avl-receiver/internal/types"
	"go.uber.org/zap"
)

type RemoteRpcStore struct {
	ProcessChan       chan types.DeviceStatus // TODO: change to a more specific type
	ResponseChan      chan types.DeviceResponse
	CloseChan         chan bool
	RemoteStoreClient CustomAvlDataStoreClient
	DeviceIdentifier  string
}

func (s *RemoteRpcStore) GetProcessChan() chan types.DeviceStatus {
	return s.ProcessChan
}

func (s *RemoteRpcStore) GetResponseChan() chan types.DeviceResponse {
	return s.ResponseChan
}

func (s *RemoteRpcStore) GetCloseChan() chan bool {
	return s.CloseChan
}

func (s *RemoteRpcStore) Process() {
	for {
		select {
		case deviceStatus := <-s.ProcessChan:
			logger.Sugar().Infoln(deviceStatus.String())
			ctx := context.Background()
			_, err := s.RemoteStoreClient.SaveDeviceStatus(ctx, &deviceStatus)
			if err != nil {
				logger.Error("failed to save device status", zap.String("imei", deviceStatus.Imei), zap.Error(err))
			}
		case <-s.CloseChan:
			logger.Sugar().Info("async remote rpc store shutting down for device")
			return
		}
	}
}

func (s *RemoteRpcStore) Response() {
	for {
		select {
		case deviceResponse := <-s.ResponseChan:
			logger.Sugar().Info(deviceResponse.String())
			start := time.Now()
			ctx := context.Background()
			_, err := s.RemoteStoreClient.SaveDeviceResponse(ctx, &deviceResponse)
			if err != nil {
				logger.Error("failed to save device status", zap.String("imei", deviceResponse.Imei), zap.Error(err))
				duration := time.Since(start)
				logger.Sugar().Info("total time taken: ", duration.Seconds())
			}

		case <-s.CloseChan:
			logger.Sugar().Info("async remote rpc store shutting down for device")
			return
		}

	}
}
