package gt06

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Packet struct {
	StartBit                uint16
	PacketLength            int8
	MessageType             MessageType
	Information             interface{}
	InformationSerialNumber uint16
	Crc                     uint16
	StopBits                uint16
}

type LoginData struct {
	TerminalID   string
	TerminalType [2]byte
	Timezone     *time.Location
}

type HeartbeatData struct {
	TerminalInformation TerminalInformation
	BatteryLevel        BatteryLevel
	GSMSignalStrength   GSMSignalStrength
	ExtendedPortStatus  uint16 // 0x0001 Chinese, 0x0002 English
}

type PositioningInformation struct {
	GpsInformation    GPSInformation
	LBSInfo           LBSInformation
	ACCHigh           bool
	DataReportingMode GPSDataUploadMode // for concox, but not for gt06
	GPSRealTime       bool              // 0x00 - realtime, 0x01 - re-upload
	MileageStatistics uint32
}

type GPSInformation struct {
	Timestamp          time.Time
	GPSInfoLength      uint8
	NumberOfSatellites uint8
	Latitude           float32
	Longitude          float32
	Speed              uint8     // gpsSpeed
	Course             GPSCourse // course/heading - running direction of GPS
}

type GPSCourse struct {
	IsRealtime     bool
	IsDifferential bool
	Positioned     bool
	Longitude      bool // 0 - east, 1 - west
	Latitude       bool // 1 - north, 0 - south
	Degree         uint16
}

type LBSInformation struct {
	MCC    uint16 // mobile country code
	MNC    uint8  // mobile network code
	LAC    uint16 // location area code
	CellID [3]byte
}

type AlarmInformation struct {
	GpsInformation    GPSInformation
	LBSInformation    LBSInformation
	StatusInformation StatusInformation
}

type StatusInformation struct {
	TerminalInformation TerminalInformation
	BatteryLevel        BatteryLevel
	GSMSignalStrength   GSMSignalStrength
	AlarmStatus         uint16
}

type TerminalInformation struct {
	OilElectricityConnected bool
	GPSSignalAvailable      bool
	AlarmType               AlarmType // concox has no alarm type, while gt06 does
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
	_ = binary.Write(&b, binary.BigEndian, r.StartBit)
	_ = binary.Write(&b, binary.BigEndian, r.PacketLength)
	_ = binary.Write(&b, binary.BigEndian, r.ProtocolNumber)
	_ = binary.Write(&b, binary.BigEndian, r.InformationSerialNumber)
	_ = binary.Write(&b, binary.BigEndian, r.Crc)
	_ = binary.Write(&b, binary.BigEndian, r.StopBits)
	return b.Bytes()
}

type MessageType byte

const (
	MSG_LoginData               MessageType = 0x01
	MSG_PositioningData                     = 0x22
	MSG_HeartbeatData                       = 0x13
	MSG_StringInformation                   = 0x21
	MSG_AlarmData                           = 0x26
	MSG_LBSInformation                      = 0x28 // TODO: check if this is correct
	MSG_TimezoneInformation                 = 0x27
	MSG_GPS_PhoneNumber                     = 0x2a
	MSG_WifiInformation                     = 0x2c
	MSG_TransmissionInstruction             = 0x80
	MSG_Invalid                             = 0xff
)

func MessageTypeFromId(id byte) MessageType {
	// write switch cases to create message type from byte id
	switch id {
	case 0x01:
		return MSG_LoginData
	case 0x22:
		return MSG_PositioningData
	case 0x13:
		return MSG_HeartbeatData
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

type AlarmType uint8 // alarm type is 3 bit info, trying to encode it to 8 bit

const (
	AL_SOSDistress  AlarmType = 0x04 // 100 -> 0000 0100
	AL_LowBattery             = 0x03 // 011 -> 0000 0011
	AL_PowerFailure           = 0x02 // 010 -> 0000 0010
	AL_Vibration              = 0x01 // 001 -> 0000 0001
	AL_Normal                 = 0x00 // 000 -> 0000 0000
	AL_Invalid                = 0xff
)

func AlarmTypeFromId(id byte) AlarmType {
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

type BatteryLevel uint8

const (
	VL_NoPower             BatteryLevel = 0x00
	VL_BatteryExtremelyLow              = 0x01
	VL_BatteryVeryLow                   = 0x02
	VL_BatteryLow                       = 0x03
	VL_BatteryMedium                    = 0x04
	VL_BatteryHigh                      = 0x05
	VL_BatteryFull                      = 0x06
	VL_Invalid                          = 0xff
)

func BatteryLevelFromByte(b byte) BatteryLevel {
	switch b {
	case 0x00:
		return VL_NoPower
	case 0x01:
		return VL_BatteryExtremelyLow
	case 0x02:
		return VL_BatteryVeryLow
	case 0x03:
		return VL_BatteryLow
	case 0x04:
		return VL_BatteryMedium
	case 0x05:
		return VL_BatteryHigh
	case 0x06:
		return VL_BatteryFull
	default:
		return VL_Invalid
	}
}

type GSMSignalStrength uint8

const (
	GSM_NoSignal            GSMSignalStrength = 0x00
	GSM_ExtremelyWeakSignal                   = 0x01
	GSM_WeakSignal                            = 0x02
	GSM_GoodSignal                            = 0x03
	GSM_StrongSignal                          = 0x04
	GSM_Invalid                               = 0xff
)

func GSMSignalStrengthFromByte(b byte) GSMSignalStrength {
	switch b {
	case 0x00:
		return GSM_NoSignal
	case 0x01:
		return GSM_ExtremelyWeakSignal
	case 0x02:
		return GSM_WeakSignal
	case 0x03:
		return GSM_GoodSignal
	case 0x04:
		return GSM_StrongSignal
	default:
		return GSM_Invalid
	}
}

type GPSDataUploadMode uint8

func (m *GPSDataUploadMode) ToString() string {
	switch *m {
	case 0x00:
		return "Upload by time interval"
	case 0x01:
		return "Upload by distance interval"
	case 0x02:
		return "Inflection point upload"
	case 0x03:
		return "ACC Status upload"
	case 0x04:
		return "Re-upload the last GPS point when back to static"
	case 0x05:
		return "Upload the last effective point when network recovers"
	case 0x06:
		return "Update ephemeris and upload GPS data compulsorily"
	case 0x07:
		return "Upload location when side key triggered"
	case 0x08:
		return "Upload location after power on"
	case 0x09:
		return "Unused"
	case 0x0A:
		return "Upload the last longitude and latitude when device is static; time updated"
	case 0x0D:
		return "Upload the last longitude and latitude when device is static"
	case 0x0E:
		return "Gps dup upload (upload regularly in static state)"
	default:
		return "Invalid"
	}
}

func (packet *Packet) ToProtobufDeviceStatus(imei string, deviceType types.DeviceType) *types.DeviceStatus {
	info := &types.DeviceStatus{}

	info.Imei = imei
	info.DeviceType = deviceType
	info.Timestamp = timestamppb.New(time.Now())
	info.VehicleStatus = &types.VehicleStatus{}
	info.Position = &types.GPSPosition{}

	// location info
	switch v := packet.Information.(type) {
	case *PositioningInformation:
	case *AlarmInformation:
		info.Timestamp = timestamppb.New(v.GpsInformation.Timestamp)
		info.Position.Latitude = v.GpsInformation.Latitude
		info.Position.Longitude = v.GpsInformation.Longitude
		info.Position.Speed = float32(v.GpsInformation.Speed)
	default:
	}

	// vehicle status
	switch v := packet.Information.(type) {
	case *PositioningInformation:
		info.VehicleStatus.Ignition = v.ACCHigh
		info.VehicleStatus.Overspeeding = false
	case *AlarmInformation:
		info.VehicleStatus.Ignition = v.StatusInformation.TerminalInformation.ACCHigh
	case *HeartbeatData:
		info.VehicleStatus.Ignition = v.TerminalInformation.ACCHigh
	default:
	}

	rawdata, _ := json.Marshal(packet)
	info.RawData = &types.DeviceStatus_WanwayPacket{
		WanwayPacket: &types.WanwayPacket{RawData: rawdata},
	}

	return info
}
