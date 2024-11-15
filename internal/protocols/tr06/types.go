package tr06

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"time"

	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	StartBitValue = 0x7979
	StopBitValue  = 0x0D0A
)

type InformationType byte

const (
	ExternalPowerVoltage InformationType = 0x00
	TerminalStatusSync   InformationType = 0x04
	DoorStatus           InformationType = 0x05
)

type ExternalPowerVoltageData struct {
	Voltage float32
}

type TerminalStatusSyncData struct {
	Status string
}

type DoorStatusData struct {
	DoorOpen bool
}

type InformationTransmissionPacket struct {
	StartBit                uint16
	PacketLength            uint16
	ProtocolNumber          byte
	InformationContent      InformationContent
	InformationSerialNumber uint16
	Crc                     uint16
	StopBits                uint16
}

type InformationContent struct {
	InformationType InformationType
	DataContent     uint16
}

type Packet struct {
	StartBit                uint16
	PacketLength            byte
	MessageType             MessageType
	Information             interface{}
	InformationSerialNumber uint16
	Crc                     uint16
	StopBits                uint16
}

type LoginData struct {
	TerminalID string
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
	Alarm               AlarmValue
	Language            Language
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
	MSG_PositioningData                     = 0x12
	MSG_HeartbeatData                       = 0x13
	MSG_EG_HeartbeatData                    = 0x23
	MSG_StringInformation                   = 0x21
	MSG_AlarmData                           = 0x26
	MSG_LBSInformation                      = 0x28 // TODO: check if this is correct
	MSG_TimezoneInformation                 = 0x27
	MSG_GPS_PhoneNumber                     = 0x2a
	MSG_WifiInformation                     = 0x2c
	MSG_TransmissionInstruction             = 0x80
	MSG_Invalid                             = 0xff
)

type AlarmType uint8 // alarm type is 3 bit info, trying to encode it to 8 bit

const (
	AL_SOSDistress  AlarmType = 0x04 // 100 -> 0000 0100
	AL_LowBattery             = 0x03 // 011 -> 0000 0011
	AL_PowerFailure           = 0x02 // 010 -> 0000 0010
	AL_Vibration              = 0x01 // 001 -> 0000 0001
	AL_Normal                 = 0x00 // 000 -> 0000 0000
	AL_Invalid                = 0xff
)

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

type AlarmValue uint8

const (
	ALV_Normal                       = 0x00
	ALV_SOS                          = 0x01
	ALV_PowerCut                     = 0x02
	ALV_Vibration                    = 0x03
	ALV_EnterFence                   = 0x04
	ALV_ExitFence                    = 0x05
	ALV_OverSpeed                    = 0x06
	ALV_Moving                       = 0x09
	ALV_EnterGPSDeadZone             = 0x0a
	ALV_ExitGPSDeadZone              = 0x0b
	ALV_PowerOn                      = 0x0c
	ALV_GPSFirstFixNotice            = 0x0d
	ALV_ExternalLowBattery           = 0x0e
	ALV_ExternalLowBatteryProtection = 0x0f
	ALV_SIMChange                    = 0x10
	ALV_PowerOff                     = 0x11
	ALV_AirplaneMode                 = 0x12
	ALV_Diassemble                   = 0x13
	ALV_Door                         = 0x14
	ALV_ShutdownLowPower             = 0x15
	ALV_Sound                        = 0x16
	ALV_InternalBatteryLow           = 0x17
	ALV_SleepMode                    = 0x20
	ALV_HarshAcceleration            = 0x29
	ALV_HarshBraking                 = 0x30
	ALV_SharpLeftTurn                = 0x2a
	ALV_SharpRightTurn               = 0x2b
	ALV_SharpCrash                   = 0x2c
	ALV_Pull                         = 0x32
	ALV_PressToUploadAlarmMessageBtn = 0x3e
	ALV_Fall                         = 0x23
	ALV_ACCOn                        = 0xee
	ALV_ACCOff                       = 0xff
)

type Language uint8

const (
	LANG_Chinese = 0x01
	LANG_English = 0x02
	LANG_NoReply = 0x00
)

type GSMSignalStrength uint8

const (
	GSM_NoSignal            GSMSignalStrength = 0x00
	GSM_ExtremelyWeakSignal                   = 0x01
	GSM_WeakSignal                            = 0x02
	GSM_GoodSignal                            = 0x03
	GSM_StrongSignal                          = 0x04
	GSM_Invalid                               = 0xff
)

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
	info.MessageType = packet.MessageType.String()
	logger.Sugar().Info("message type: ", info.MessageType)

	// location info
	switch v := packet.Information.(type) {
	case *PositioningInformation:
		info.Timestamp = timestamppb.New(v.GpsInformation.Timestamp)
		info.Position.Latitude = v.GpsInformation.Latitude
		info.Position.Longitude = v.GpsInformation.Longitude
		info.Position.Course = float32(v.GpsInformation.Course.Degree)
		var speed = float32(v.GpsInformation.Speed)
		info.Position.Speed = &speed
	case *AlarmInformation:
		info.Timestamp = timestamppb.New(v.GpsInformation.Timestamp)
		info.Position.Latitude = v.GpsInformation.Latitude
		info.Position.Longitude = v.GpsInformation.Longitude
		var speed = float32(v.GpsInformation.Speed)
		info.Position.Speed = &speed
	default:
	}

	// vehicle status
	switch v := packet.Information.(type) {
	case *PositioningInformation:
		var ignition = v.ACCHigh
		info.VehicleStatus.Ignition = &ignition
		info.VehicleStatus.OverSpeeding = false
	case *AlarmInformation:
		var ignition = v.StatusInformation.TerminalInformation.ACCHigh
		info.VehicleStatus.Ignition = &ignition
		info.VehicleStatus.OverSpeeding = v.StatusInformation.Alarm == ALV_OverSpeed
	case *HeartbeatData:

	default:
	}

	rawdata, _ := json.Marshal(packet)
	info.RawData = &types.DeviceStatus_WanwayPacket{
		WanwayPacket: &types.WanwayPacket{RawData: rawdata},
	}

	return info
}

func (mt MessageType) String() string {
	switch mt {
	case MSG_LoginData:
		return "MSG_LoginData"
	case MSG_PositioningData:
		return "MSG_PositioningData"
	case MSG_HeartbeatData:
		return "MSG_HeartbeatData"
	case MSG_EG_HeartbeatData:
		return "MSG_EG_HeartbeatData"
	case MSG_StringInformation:
		return "MSG_StringInformation"
	case MSG_AlarmData:
		return "MSG_AlarmData"
	case MSG_LBSInformation:
		return "MSG_LBSInformation"
	case MSG_TimezoneInformation:
		return "MSG_TimezoneInformation"
	case MSG_GPS_PhoneNumber:
		return "MSG_GPS_PhoneNumber"
	case MSG_WifiInformation:
		return "MSG_WifiInformation"
	case MSG_TransmissionInstruction:
		return "MSG_TransmissionInstruction"
	default:
		return "MSG_Invalid"
	}
}
