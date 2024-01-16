package handlers

import (
	"github.com/404minds/avl-receiver/internal/devices"
)

var TcpHandler = tcpHandler{
	connToProtocolMap: make(map[string]devices.DeviceProtocol),
	// registered device types can be made configurable to enable/disable a device-type at once
	registeredDeviceTypes: []devices.AVLDeviceType{devices.Teltonika},
}
