package wanway

import (
	"bytes"
	"encoding/binary"
	"time"
)

type WanwayPacket struct {
	StartBit                uint16
	PacketLength            int8
	MessageType             WanwayMessageType
	Information             interface{}
	InformationSerialNumber uint16
	Crc                     uint16
	StopBits                uint16
}

type WanwayLoginInformation struct {
	TerminalID   string
	TerminalType [2]byte
	Timezone     *time.Location
}

type WanwayPositioningInformation struct {
	GPSInfo           WanwayGPSInformation
	LBSInfo           WanwayLBSInformation
	ACC               byte
	DataReportingMode byte
	RealtimeGPSPassUp byte
	MileageStatistics uint32
}

type WanwayGPSInformation struct {
	Timestamp          time.Time
	GPSInfoLength      uint8
	NumberOfSatellites uint8
	Latitude           uint32
	Longitude          uint32
	Speed              uint8  // gpsSpeed
	Course             uint16 // course/heading - running direction of GPS
}

type WanwayLBSInformation struct {
	MCC    uint16 // mobile country code
	MNC    uint8  // mobile network code
	LAC    uint16 // location area code
	CellID [3]byte
}

type WanwayAlarmInformation struct {
	GpsInformation    WanwayGPSInformation
	LBSInformation    WanwayLBSInformation
	StatusInformation WanwayStatusInformation
}

type WanwayStatusInformation struct {
	TerminalInformation WanwayTerminalInformation
	VoltageLevel        uint8
	GSMSignalStrength   uint8
	AlarmStatus         uint16
}

type WanwayTerminalInformation struct {
	OilElectricityConnected bool
	GPSSignalAvailable      bool
	AlarmType               WanwayAlarmType
	Charging                bool
	ACCHigh                 bool
	Armed                   bool
}

type ResponsePacket struct {
	StartBit                uint16
	PacketLength            int8
	ProtocolNumber          int8
	InformationSerialNumber uint16
	Crc                     uint16
	StopBits                uint16
}

func (r *ResponsePacket) ToBytes() []byte {
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, r.StartBit)
	binary.Write(&b, binary.BigEndian, r.PacketLength)
	binary.Write(&b, binary.BigEndian, r.ProtocolNumber)
	binary.Write(&b, binary.BigEndian, r.InformationSerialNumber)
	binary.Write(&b, binary.BigEndian, r.Crc)
	binary.Write(&b, binary.BigEndian, r.StopBits)
	return b.Bytes()
}

type WanwayMessageType byte

const (
	MSG_LoginInformation        WanwayMessageType = 0x01
	MSG_PositioningData                           = 0x22
	MSG_StatusInformation                         = 0x13
	MSG_StringInformation                         = 0x21
	MSG_LBSInformation                            = 0x2d // TODO: check if this is correct
	MSG_AlarmData                                 = 0x26
	MSG_TimezoneInformation                       = 0x27
	MSG_GPS_PhoneNumber                           = 0x2a
	MSG_WifiInformation                           = 0x2c
	MSG_TransmissionInstruction                   = 0x80
	MSG_Invalid                                   = 0xff
)

func WanwayMessageTypeFromId(id byte) WanwayMessageType {
	// write switch cases to create message type from byte id
	switch id {
	case 0x01:
		return MSG_LoginInformation
	case 0x22:
		return MSG_PositioningData
	case 0x13:
		return MSG_StatusInformation
	case 0x21:
		return MSG_StringInformation
	case 0x2d: // TODO: check if this is correct
		return MSG_LBSInformation
	case 0x26:
		return MSG_AlarmData
	case 0x27:
		return MSG_TimezoneInformation
	case 0x2a:
		return MSG_GPS_PhoneNumber
	case 0x2c:
		return MSG_WifiInformation
	case 0x80:
		return MSG_TransmissionInstruction
	default:
		return MSG_Invalid
	}
}

type WanwayAlarmType uint8 // alarm type is 3 bit info, trying to encode it to 8 bit

const (
	AL_SOSDistress  WanwayAlarmType = 0x04 // 100 -> 0000 0100
	AL_LowBattery                   = 0x03 // 011 -> 0000 0011
	AL_PowerFailure                 = 0x02 // 010 -> 0000 0010
	AL_Vibration                    = 0x01 // 001 -> 0000 0001
	AL_Normal                       = 0x00 // 000 -> 0000 0000
	AL_Invalid                      = 0xff
)

func WanwayAlarmTypeFromId(id byte) WanwayAlarmType {
	switch id {
	case 0x04:
		return AL_SOSDistress
	case 0x03:
		return AL_LowBattery
	case 0x02:
		return AL_PowerFailure
	case 0x01:
		return AL_Vibration
	case 0x00:
		return AL_Normal
	default:
		return AL_Invalid
	}
}
