package errors

import "errors"

var ErrUnknownDeviceType = errors.New("unknown device type")
var ErrTeltonikaUnauthorizedDevice = errors.New("unauthorized teltonika device")
var ErrTeltonikaInvalidDataPacket = errors.New("invalid teltonika data packet")
var ErrTeltonikaBadCrc = errors.New("bad crc in teltonika data packet")
