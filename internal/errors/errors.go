package errors

import (
	"errors"
	"fmt"
)

var ErrUnknownProtocol = errors.New("unknown protocol")
var ErrUnknownDeviceType = errors.New("unknown device type")
var ErrUnauthorizedDevice = errors.New("unauthorized device")
var ErrBadCrc = errors.New("bad crc")
var ErrBadPacket = errors.New("bad data packet")
var ErrSendingResponse = errors.New("error while sending response packet")

var ErrFM1200BadDataPacket = fmt.Errorf("bad fm1200 data packet: %w", ErrBadPacket)
var ErrTR06BadDataPacket = errors.New("invalid tr06 data packet")
var ErrGT06BadDataPacket = errors.New("invalid gt06 data packet")
var ErrGT06InvalidLoginInfo = errors.New("invalid gt06 login info")
var ErrTR06InvalidLoginInfo = errors.New("invalid tr06 login info")
var ErrGT06InvalidAlarmType = errors.New("invalid gt06 alarm type")
var ErrTR06InvalidAlarmType = errors.New("invalid tr06 alarm type")
var ErrGT06InvalidVoltageLevel = errors.New("invalid voltage level")
var ErrTR06InvalidVoltageLevel = errors.New("invalid voltage level")
var ErrGT06InvalidGSMSignalStrength = errors.New("invalid gsm signal strength")
var ErrTR06InvalidGSMSignalStrength = errors.New("invalid gsm signal strength")
