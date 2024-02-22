package devices

import (
	"github.com/404minds/avl-receiver/internal/devices/teltonika"
	"github.com/404minds/avl-receiver/internal/devices/wanway"
	"github.com/404minds/avl-receiver/internal/types"
)

func GetProtocolForDeviceType(t types.DeviceType) DeviceProtocol {
	switch t {
	case types.DeviceType_Teltonika:
		return &teltonika.TeltonikaProtocol{}
	case types.DeviceType_Wanway:
	case types.DeviceType_Concox:
		return &wanway.WanwayProtocol{}
	default:
		return nil
	}
}
