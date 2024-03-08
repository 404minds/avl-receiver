package devices

import (
	"github.com/404minds/avl-receiver/internal/devices/teltonika"
	"github.com/404minds/avl-receiver/internal/devices/wanway"
	"github.com/404minds/avl-receiver/internal/types"
)

func MakeProtocolForType(t types.DeviceProtocolType) DeviceProtocol {
	switch t {
	case types.DeviceProtocolType_FM1200:
		return &teltonika.TeltonikaProtocol{}
	case types.DeviceProtocolType_GT06:
		return &wanway.WanwayProtocol{}
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
	default:
		return []types.DeviceType{}
	}
}
