package devices

import (
	"github.com/404minds/avl-receiver/internal/devices/teltonika"
	"github.com/404minds/avl-receiver/internal/devices/wanway"
)

// enum for device types
type AVLDeviceType int

const (
	Teltonika AVLDeviceType = iota + 1
	Wanway
)

func (t AVLDeviceType) String() string {
	return [...]string{"Teltonika", "Wanway"}[t-1]
}

func (t AVLDeviceType) GetProtocol() DeviceProtocol {
	switch t {
	case Teltonika:
		return &teltonika.TeltonikaProtocol{}
	case Wanway:
		return &wanway.WanwayProtocol{}
	default:
		return nil
	}
}
