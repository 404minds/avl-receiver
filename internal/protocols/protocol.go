package protocols

import (
	"bufio"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/protocols/fm1200"
	"github.com/404minds/avl-receiver/internal/protocols/gt06"
	"github.com/404minds/avl-receiver/internal/protocols/tr06"
	"io"

	"github.com/404minds/avl-receiver/internal/types"
)

var logger = configuredLogger.Logger

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
		logger.Sugar().Info("GT06 called the protocol is: ", &gt06.GT06Protocol{})
		return &gt06.GT06Protocol{}

	case types.DeviceProtocolType_TR06:
		logger.Sugar().Info("TR06 called the protocol is: ", &tr06.TR06Protocol{})
		return &tr06.TR06Protocol{}

	default:

		logger.Sugar().Info("MakeProtocolForType: ", t)
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
