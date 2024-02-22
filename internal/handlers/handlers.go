package handlers

import (
	"github.com/404minds/avl-receiver/internal/devices"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
)

func NewTcpHandler(datadir string) tcpHandler {
	return tcpHandler{
		connToProtocolMap:     make(map[string]devices.DeviceProtocol),
		registeredDeviceTypes: []types.DeviceType{types.DeviceType_Teltonika, types.DeviceType_Wanway, types.DeviceType_Concox}, // registered device types can be made configurable to enable/disable a device-type at once
		connToStoreMap:        make(map[string]store.Store),
		dataDir:               datadir,
	}

}
