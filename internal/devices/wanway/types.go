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
	Speed              uint8 // gpsSpeed
	Course             uint16
}

type WanwayLBSInformation struct {
	MCC    uint16 // mobile country code
	MNC    uint8  // mobile network code
	LAC    uint16 // location area code
	CellID [3]byte
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
	MSG_LBSInformation                            = 0x22
	MSG_AlarmData                                 = 0x26
	MSG_TimezoneInformation                       = 0x27
	MSG_GPS_PhoneNumber                           = 0x2a
	MSG_WifiInformation                           = 0x2c
	MSG_TransmissionInstruction                   = 0x80
)

var wanwayMessageTypes = []WanwayMessageType{MSG_LoginInformation, MSG_PositioningData, MSG_StatusInformation, MSG_StringInformation, MSG_LBSInformation, MSG_AlarmData, MSG_TimezoneInformation, MSG_GPS_PhoneNumber, MSG_WifiInformation, MSG_TransmissionInstruction}

func WanwayMessageTypeFromId(id byte) *WanwayMessageType {
	for _, msgType := range wanwayMessageTypes {
		if byte(msgType) == id {
			return &msgType
		}
	}
	return nil
}
