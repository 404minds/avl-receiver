package devices

import "github.com/404minds/avl-receiver/internal/devices/teltonika"

// enum for device types
type AVLDeviceType int

const (
	Teltonika AVLDeviceType = iota + 1
)

func (t AVLDeviceType) String() string {
	return [...]string{"Teltonika"}[t-1]
}

func (t AVLDeviceType) GetProtocol() DeviceProtocol {
	switch t {
	case Teltonika:
		return &teltonika.TeltonikaProtocol{}
	default:
		return nil
	}
}
