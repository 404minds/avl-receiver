package handlers

import (
	"github.com/404minds/avl-receiver/internal/devices"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
)

func NewTcpHandler(remoteStoreClient store.AvlDataStoreClient) tcpHandler {
	return tcpHandler{
		connToProtocolMap:     make(map[string]devices.DeviceProtocol),
		registeredDeviceTypes: []types.DeviceType{types.DeviceType_TELTONIKA, types.DeviceType_WANWAY}, // registered device types can be made configurable to enable/disable a device-type at once
		connToStoreMap:        make(map[string]store.Store),
		remoteStoreClient:     remoteStoreClient,
	}

}
