package devices

import (
	"bufio"
	"io"

	"github.com/404minds/avl-receiver/internal/types"
)

type DeviceProtocol interface {
	GetDeviceType() types.DeviceType
	SetDeviceType(types.DeviceType)
	GetProtocolType() types.DeviceProtocolType
	GetDeviceIdentifier() string
	Login(*bufio.Reader) ([]byte, int, error)
	ConsumeStream(*bufio.Reader, io.Writer, chan types.DeviceStatus) error
}
