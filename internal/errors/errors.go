package errors

import (
	"errors"
	"fmt"
)

var ErrUnknownDeviceType = errors.New("unknown device type")
var ErrUnauthorizedDevice = errors.New("unauthorized device")

var ErrNotTeltonikaDevice = fmt.Errorf("not a teltonika device - %w", ErrUnknownDeviceType)
var ErrTeltonikaUnauthorizedDevice = fmt.Errorf("unauthorized teltonika device", ErrUnauthorizedDevice)
var ErrTeltonikaInvalidDataPacket = errors.New("invalid teltonika data packet")
var ErrTeltonikaBadCrc = errors.New("crc check failed for teltonika device")

var ErrNotWanwayDevice = fmt.Errorf("not a wanway device - %w", ErrUnknownDeviceType)
var ErrWanwayInvalidPacket = errors.New("invalid wanway data packet")
var ErrWanwayInvalidLoginInfo = errors.New("invalid wanway login info")
var ErrWanwayBadCrc = errors.New("crc check failed for wanway device")
var ErrWanwayUnauthorizedDevice = fmt.Errorf("unauthorized wanway device", ErrUnauthorizedDevice)
var ErrWanwayInvalidAlarmType = errors.New("invalid wanway alarm type")
var ErrWanwayInvalidVoltageLevel = errors.New("invalid voltage level")
var ErrWanwayInvalidGSMSignalStrength = errors.New("invalid gsm signal strength")
