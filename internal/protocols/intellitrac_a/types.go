package intellitrac_a

import (
	"time"

	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Protocol constants
const (
	PreambleSize             = 4
	BinaryHeartbeatSize      = 22
	ASCIIHeartbeatSize       = 8
	BinaryAckSize            = 6
	BinaryPositionHeaderSize = 12
)

// Message types
const (
	MsgEncodingBinaryPos  = 0x00
	MsgEncodingText       = 0x02
	MsgEncodingGarminText = 0x03
	MsgEncodingATCommand  = 0x01

	MsgTypeRequest  = 0x00
	MsgTypeResponse = 0x01
	MsgTypeAsync    = 0x02
	MsgTypeAck      = 0x03
)

// I/O Status bits (from page 11)
const (
	IgnitionStatus = 0
	Input1Status   = 1
	Input2Status   = 2
	Input3Status   = 3
	Input4Status   = 4
	Output1Status  = 8
	Output2Status  = 9
	Output3Status  = 10
)

// Vehicle Status bits (from page 11)
const (
	EngineStatus = 0
	MotionStatus = 1
)

type Heartbeat struct {
	TransactionID uint16
	ModemID       uint64
	MessageID     uint16
	DataLength    uint16
	RTC           time.Time
}

type PositionRecord struct {
	TransactionID     uint16
	ModemID           string
	MessageID         uint16
	DataLength        uint16
	GPS               GPSData
	Odometer          uint32
	HDOP              uint8
	Satellites        uint8
	IOStatus          uint16
	VehicleStatus     uint8
	AnalogInput1      float32
	AnalogInput2      float32
	RTC               time.Time
	PositionSending   time.Time
	EventData         []byte
	RawData           string
	AdditionalMetrics map[uint16]interface{}
}

type GPSData struct {
	Timestamp time.Time
	Latitude  float64
	Longitude float64
	Altitude  int32
	Speed     float32
	Direction float32
}

type IntelliTracAProtocol struct {
	Imei       string
	DeviceType types.DeviceType
	IsBinary   bool
}

func (p *PositionRecord) ToDeviceStatus(imei string) *types.DeviceStatus {
	status := &types.DeviceStatus{
		Imei:       imei,
		DeviceType: types.DeviceType_INTELLITRAC,
		Timestamp:  timestamppb.New(p.GPS.Timestamp),
		Position: &types.GPSPosition{
			Latitude:   float32(p.GPS.Latitude),
			Longitude:  float32(p.GPS.Longitude),
			Altitude:   float32(p.GPS.Altitude),
			Speed:      &p.GPS.Speed,
			Course:     float32(p.GPS.Direction),
			Satellites: int32(p.Satellites),
		},
		VehicleStatus: &types.VehicleStatus{
			Ignition: boolPtr(p.IOStatus&(1<<IgnitionStatus) != 0),
		},
		Odometer: int32(p.Odometer), // Odometer is in meters
		RawData: &types.DeviceStatus_IntellitracPacket{
			IntellitracPacket: &types.IntellitracPacket{
				RawData: []byte(p.RawData),
			},
		},
	}

	return status
}

func boolPtr(b bool) *bool {
	return &b
}

func parseDateTime(year, month, day, hour, minute, second uint8) time.Time {
	return time.Date(2000+int(year), time.Month(month), int(day),
		int(hour), int(minute), int(second), 0, time.UTC)
}
