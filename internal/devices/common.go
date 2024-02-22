package devices

import (
	"bufio"

	"github.com/404minds/avl-receiver/internal/types"
)

type DeviceProtocol interface {
	// GetDeviceType() AVLDeviceType
	GetDeviceIdentifier() string
	Login(*bufio.Reader) ([]byte, int, error)
	ConsumeStream(*bufio.Reader, *bufio.Writer, chan types.DeviceStatus) error
}
