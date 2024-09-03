package fm1200

import (
	"encoding/json"
	"time"

	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CodecID byte

const (
	Codec8  CodecID = 0x08
	Codec8E CodecID = 0x8E
)

type Record struct {
	IMEI   string    `json:"imei"`
	Record AvlRecord `json:"record"`
}

type AvlDataPacket struct {
	CodecID      uint8       `json:"codec_id"`
	NumberOfData uint8       `json:"number_of_data"`
	Data         []AvlRecord `json:"data"`
	CRC          uint32      `json:"crc"`
}

type AvlRecord struct {
	Timestamp  uint64     `json:"timestamp"`
	Priority   uint8      `json:"priority"`
	GPSElement GpsElement `json:"gps_element"`
	IOElement  IOElement  `json:"io_element"`
}

type GpsElement struct {
	Longitude  float32 `json:"longitude"`
	Latitude   float32 `json:"latitude"`
	Altitude   uint16  `json:"altitude"`
	Angle      uint16  `json:"angle"`
	Satellites uint8   `json:"satellites"`
	Speed      uint16  `json:"speed"`
}

type IOElement struct {
	EventID       uint16 `json:"event_id"`
	NumProperties uint16 `json:"num_properties"`

	Properties1B  map[IOProperty]uint8  `json:"properties_1b"`
	Properties2B  map[IOProperty]uint16 `json:"properties_2b"`
	Properties4B  map[IOProperty]uint32 `json:"properties_4b"`
	Properties8B  map[IOProperty]uint64 `json:"properties_8b"`
	PropertiesNXB map[IOProperty][]byte `json:"properties_xb"`
}

type IOProperty int

const (
	TIO_DigitalInput1      IOProperty = 1
	TIO_DigitalInput2                 = 2
	TIO_DigitalInput3                 = 3
	TIO_AnalogInput                   = 9
	TIO_PCBTemperature                = 70
	TIO_DigitalOutput1                = 179
	TIO_DigitalOutput2                = 180
	TIO_GPSPDOP                       = 181
	TIO_GPSHDOP                       = 182
	TIO_ExternalVoltage               = 66
	TIO_GPSPower                      = 69
	TIO_MovementSensor                = 240
	TIO_OdometerValue                 = 16
	TIO_FuelLevel                     = 201
	TIO_TripOdometerValue             = 199
	TIO_GSMOperator                   = 241
	TIO_Speed                         = 24
	TIO_IButtonID                     = 78
	TIO_WorkingMode                   = 80
	TIO_GSMSignal                     = 21
	TIO_SleepMode                     = 200
	TIO_CellID                        = 205
	TIO_AreaCode                      = 206
	TIO_DallasTemperature             = 72
	TIO_BatteryVoltage                = 67
	TIO_BatteryCurrent                = 68
	TIO_AutoGeofence                  = 175
	TIO_Geozone1                      = 155
	TIO_Geozone2                      = 156
	TIO_Geozone3                      = 157
	TIO_Geozone4                      = 158
	TIO_Geozone5                      = 159
	TIO_TripMode                      = 250
	TIO_Immobilizer                   = 251
	TIO_AuthorizedDriving             = 252
	TIO_GreenDrivingStatus            = 253
	TIO_GreenDrivingValue             = 254
	TIO_Overspeeding                  = 255
)

func (r *Record) ToProtobufDeviceStatus() *types.DeviceStatus {
	info := &types.DeviceStatus{}

	info.Imei = r.IMEI
	info.DeviceType = types.DeviceType_TELTONIKA
	info.Timestamp = timestamppb.New(time.Unix(int64(r.Record.Timestamp), 0))
	info.MessageType = "MSG_PositioningData"
	logger.Sugar().Info("message type: ", info.MessageType)

	// gps info
	info.Position = &types.GPSPosition{}
	info.Position.Latitude = r.Record.GPSElement.Latitude
	info.Position.Longitude = r.Record.GPSElement.Longitude
	info.Position.Altitude = float32(r.Record.GPSElement.Altitude)
	logger.Sugar().Info("parsed speed ", r.Record.GPSElement.Speed)
	var speed = float32(r.Record.GPSElement.Speed)
	info.Position.Speed = &speed
	info.Odometer = int32(r.Record.IOElement.Properties4B[TIO_OdometerValue])
	info.Position.Course = float32(r.Record.GPSElement.Angle)
	info.Position.Satellites = int32(r.Record.IOElement.Properties1B[TIO_GSMSignal])
	info.Temperature = float32(r.Record.IOElement.Properties4B[TIO_DallasTemperature])
	info.FuelLevel = int32(r.Record.IOElement.Properties2B[TIO_FuelLevel])
	// vehicle info
	info.VehicleStatus = &types.VehicleStatus{}
	var ignition = r.Record.IOElement.Properties1B[TIO_DigitalInput1] > 0
	info.VehicleStatus.Ignition = &ignition
	info.VehicleStatus.Overspeeding = r.Record.IOElement.Properties1B[TIO_Overspeeding] > 0
	info.VehicleStatus.RashDriving = r.Record.IOElement.Properties1B[TIO_GreenDrivingStatus] > 0

	//battery level
	info.BatteryLevel = int32(r.Record.IOElement.Properties2B[TIO_BatteryVoltage])

	rawdata, _ := json.Marshal(r)
	info.RawData = &types.DeviceStatus_TeltonikaPacket{
		TeltonikaPacket: &types.TeltonikaPacket{RawData: rawdata},
	}
	return info
}
