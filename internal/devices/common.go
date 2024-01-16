package devices

import "bufio"

type DeviceProtocol interface {
	// GetDeviceType() AVLDeviceType
	GetDeviceIdentifier() string
	Login(*bufio.Reader) ([]byte, int, error)
	ConsumeStream(*bufio.Reader, *bufio.Writer, chan interface{}) error
}
