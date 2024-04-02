package handlers

import (
	"github.com/404minds/avl-receiver/internal/protocols"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
)

func NewTcpHandler(remoteStoreClient store.AvlDataStoreClient, storeType string) TcpHandler {
	return TcpHandler{
		connToProtocolMap: make(map[string]protocols.DeviceProtocol),
		allowedProtocols:  []types.DeviceProtocolType{types.DeviceProtocolType_FM1200, types.DeviceProtocolType_GT06}, // registered device types can be made configurable to enable/disable a device-type at once
		connToStoreMap:    make(map[string]store.Store),
		remoteStoreClient: remoteStoreClient,
		storeType:         storeType,
	}
}
