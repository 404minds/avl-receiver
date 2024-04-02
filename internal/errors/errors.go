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

var ErrGT06BadDataPacket = errors.New("invalid gt06 data packet")
var ErrGT06InvalidLoginInfo = errors.New("invalid gt06 login info")
var ErrGT06InvalidAlarmType = errors.New("invalid gt06 alarm type")
var ErrGT06InvalidVoltageLevel = errors.New("invalid voltage level")
var ErrGT06InvalidGSMSignalStrength = errors.New("invalid gsm signal strength")
