package handlers

import (
	"github.com/404minds/avl-receiver/internal/devices"
	"github.com/404minds/avl-receiver/internal/store"
)

var TcpHandler = tcpHandler{
	connToProtocolMap:     make(map[string]devices.DeviceProtocol),
	registeredDeviceTypes: []devices.AVLDeviceType{devices.Teltonika}, // registered device types can be made configurable to enable/disable a device-type at once
	connToStoreMap:        make(map[string]store.Store),
}
