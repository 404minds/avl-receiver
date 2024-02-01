package errors

import (
	"errors"
	"fmt"
)

var ErrUnknownDeviceType = errors.New("unknown device type")

var ErrNotTeltonikaDevice = fmt.Errorf("not a teltonika device - %w", ErrUnknownDeviceType)
var ErrTeltonikaUnauthorizedDevice = errors.New("unauthorized teltonika device")
var ErrTeltonikaInvalidDataPacket = errors.New("invalid teltonika data packet")
var ErrTeltonikaBadCrc = errors.New("crc check failed for teltonika device")

var ErrNotWanwayDevice = fmt.Errorf("not a wanway device - %w", ErrUnknownDeviceType)
var ErrWanwayInvalidPacket = errors.New("invalid wanway data packet")
var ErrWanwayInvalidLoginInfo = errors.New("invalid wanway login info")
var ErrWanwayBadCrc = errors.New("crc check failed for wanway device")
var ErrWanwayUnauthorizedDevice = errors.New("unauthorized wanway device")
var ErrWanwayInvalidAlarmType = errors.New("invalid wanway alarm type")
