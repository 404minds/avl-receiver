package protocols

import (
	"bufio"
	"github.com/404minds/avl-receiver/internal/protocols/fm1200"
	"github.com/404minds/avl-receiver/internal/protocols/gt06"
	"github.com/404minds/avl-receiver/internal/protocols/tr06"
	"io"

	"github.com/404minds/avl-receiver/internal/types"
)

type DeviceProtocol interface {
	GetDeviceType() types.DeviceType
	SetDeviceType(types.DeviceType)
	GetProtocolType() types.DeviceProtocolType
	GetDeviceID() string
	Login(*bufio.Reader) ([]byte, int, error)
	ConsumeStream(*bufio.Reader, io.Writer, chan types.DeviceStatus) error
}

func MakeProtocolForType(t types.DeviceProtocolType) DeviceProtocol {
	switch t {
	case types.DeviceProtocolType_FM1200:
		return &fm1200.FM1200Protocol{}

	case types.DeviceProtocolType_GT06:
		return &gt06.GT06Protocol{}

	case types.DeviceProtocolType_TR06:
		return &tr06.TR06Protocol{}

	default:
		return nil
	}

}

func GetDeviceTypesForProtocol(t types.DeviceProtocolType) []types.DeviceType {
	switch t {
	case types.DeviceProtocolType_FM1200:
		return []types.DeviceType{types.DeviceType_TELTONIKA}
	case types.DeviceProtocolType_GT06:
		return []types.DeviceType{types.DeviceType_CONCOX, types.DeviceType_WANWAY}
	case types.DeviceProtocolType_TR06:
		return []types.DeviceType{types.DeviceType_CONCOX, types.DeviceType_WANWAY}
	default:
		return []types.DeviceType{}
	}
}
