package store

import (
	"context"
	"time"

	"github.com/404minds/avl-receiver/internal/types"
	"go.uber.org/zap"
)

type RemoteRpcStore struct {
	ProcessChan       chan types.DeviceStatus // TODO: change to a more specific type
	CloseChan         chan bool
	RemoteStoreClient AvlDataStoreClient
	DeviceIdentifier  string
}

func (s *RemoteRpcStore) GetProcessChan() chan types.DeviceStatus {
	return s.ProcessChan
}

func (s *RemoteRpcStore) GetCloseChan() chan bool {
	return s.CloseChan
}

func (s *RemoteRpcStore) Process() {
	for {
		select {
		case deviceStatus := <-s.ProcessChan:
			logger.Sugar().Infoln(deviceStatus.String())
			ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
			_, err := s.RemoteStoreClient.SaveDeviceStatus(ctx, &deviceStatus)
			if err != nil {
				logger.Error("failed to save device status", zap.String("imei", deviceStatus.Imei), zap.Error(err))
			}
		case <-s.CloseChan:
			logger.Sugar().Info("async remote rpc store shutting down for device id", s.DeviceIdentifier)
			return
		}
	}
}
