package devices

import (
	"github.com/404minds/avl-receiver/internal/devices/teltonika"
	"github.com/404minds/avl-receiver/internal/devices/wanway"
	"github.com/404minds/avl-receiver/internal/types"
)

func MakeProtocolForDeviceType(t types.DeviceType) DeviceProtocol {
	switch t {
	case types.DeviceType_TELTONIKA:
		return &teltonika.TeltonikaProtocol{}
	case types.DeviceType_WANWAY:
		return &wanway.WanwayProtocol{}
	default:
		return nil
	}
	return nil
}
