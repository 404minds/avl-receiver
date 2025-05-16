package protocols

import (
	"bufio"
	"io"

	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/protocols/fm1200"
	"github.com/404minds/avl-receiver/internal/protocols/gt06"
	"github.com/404minds/avl-receiver/internal/protocols/howen"
	"github.com/404minds/avl-receiver/internal/protocols/obdii2g"
	"github.com/404minds/avl-receiver/internal/protocols/tr06"
	"github.com/404minds/avl-receiver/internal/store"

	"github.com/404minds/avl-receiver/internal/types"
)

var logger = configuredLogger.Logger

type DeviceProtocol interface {
	GetDeviceType() types.DeviceType
	SetDeviceType(types.DeviceType)
	GetProtocolType() types.DeviceProtocolType
	GetDeviceID() string
	Login(*bufio.Reader) ([]byte, int, error)
	ConsumeStream(*bufio.Reader, io.Writer, store.Store) error
	SendCommandToDevice(writer io.Writer, command string) error
}

func MakeProtocolForType(t types.DeviceProtocolType) DeviceProtocol {
	switch t {
	case types.DeviceProtocolType_FM1200:
		return &fm1200.FM1200Protocol{}

	case types.DeviceProtocolType_GT06:
		logger.Sugar().Info("GT06 called the protocol is: ", &gt06.GT06Protocol{DeviceType: types.DeviceType_CONCOX})
		return &gt06.GT06Protocol{DeviceType: types.DeviceType_CONCOX}

	case types.DeviceProtocolType_OBDII2G:
		logger.Sugar().Info("OBDII2G called the protocol is: ", &obdii2g.AquilaOBDII2GProtocol{DeviceType: types.DeviceType_AQUILA})
		return &obdii2g.AquilaOBDII2GProtocol{DeviceType: types.DeviceType_AQUILA}

	case types.DeviceProtocolType_TR06:
		logger.Sugar().Info("TR06 called the protocol is: ", &tr06.TR06Protocol{DeviceType: types.DeviceType_WANWAY})
		return &tr06.TR06Protocol{DeviceType: types.DeviceType_WANWAY}

	case types.DeviceProtocolType_HOWENWS:
		logger.Sugar().Info("Howen called the protocol is: ", &howen.HOWENWS{DeviceType: types.DeviceType_HOWEN})
		return &howen.HOWENWS{DeviceType: types.DeviceType_HOWEN}
	default:

		logger.Sugar().Info("MakeProtocolForType: ", t)
		return nil
	}

}

func GetDeviceTypesForProtocol(t types.DeviceProtocolType) []types.DeviceType {
	switch t {
	case types.DeviceProtocolType_FM1200:
		return []types.DeviceType{types.DeviceType_TELTONIKA}
	case types.DeviceProtocolType_OBDII2G:
		return []types.DeviceType{types.DeviceType_AQUILA}
	case types.DeviceProtocolType_GT06:
		return []types.DeviceType{types.DeviceType_CONCOX, types.DeviceType_WANWAY}
	case types.DeviceProtocolType_TR06:
		return []types.DeviceType{types.DeviceType_WANWAY, types.DeviceType_CONCOX}
	case types.DeviceProtocolType_HOWENWS:
		return []types.DeviceType{types.DeviceType_HOWEN}
	default:
		return []types.DeviceType{types.DeviceType_AQUILA}
	}
}
