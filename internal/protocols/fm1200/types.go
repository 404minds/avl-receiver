package fm1200

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CodecID uint

const (
	Codec8  CodecID = 0x08
	Codec8E CodecID = 0x8E
	codec12 CodecID = 0x0C
	codec13 CodecID = 0x0D
	codec14 CodecID = 0x0E
)

type DeviceResponse struct {
	CodecID           byte   // Codec ID (always 0x0C for Codec12)
	ResponseQuantity1 byte   // Response Quantity 1
	Type              byte   // Response Type (0x06 for response)
	ResponseSize      uint32 // Response Size in bytes
	ResponseData      []byte // Actual response data (in bytes)
	ResponseQuantity2 byte   // Response Quantity 2 (should match Response Quantity 1)
	CRC               uint32 // CRC-16 checksum
}

type Response struct {
	IMEI  string
	Reply []byte
}

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
	TIO_DigitalInput1        IOProperty = 1
	TIO_DigitalInput2                   = 2
	TIO_DigitalInput3                   = 3
	TIO_AnalogInput                     = 9
	TIO_FuelUsedGPS                     = 12
	TIO_OdometerValue                   = 16
	TIO_GSMSignal                       = 21
	TIO_Speed                           = 24
	TIO_EngineLoad                      = 31
	TIO_CoolantTemperature              = 32
	TIO_RPM                             = 36
	TIO_Vehicle_Speed_OBD               = 37
	TIO_FuelLevelPCT_OBD                = 48
	TIO_AmbientTemperature              = 53
	TIO_ExternalVoltage                 = 66
	TIO_BatteryVoltage                  = 67
	TIO_BatteryCurrent                  = 68
	TIO_GPSPower                        = 69
	TIO_PCBTemperature                  = 70
	TIO_DallasTemperature               = 72
	TIO_IButtonID                       = 78
	TIO_WorkingMode                     = 80
	TIO_Speed_CAN                       = 81
	TIO_FuelConsumed_CAN                = 83
	TIO_FuelLevelLtr_CAN                = 84
	TIO_RPM_CAN                         = 85
	TIO_TotalMileage_CAN                = 87
	TIO_FuelLevelPercent_CAN            = 89
	TIO_Geozone1                        = 155
	TIO_Geozone2                        = 156
	TIO_Geozone3                        = 157
	TIO_Geozone4                        = 158
	TIO_Geozone5                        = 159
	TIO_AutoGeofence                    = 175
	TIO_DigitalOutput1                  = 179
	TIO_DigitalOutput2                  = 180
	TIO_GPSPDOP                         = 181
	TIO_GPSHDOP                         = 182
	TIO_TripOdometerValue               = 199
	TIO_SleepMode                       = 200
	TIO_CellID                          = 205
	TIO_AreaCode                        = 206
	TIO_RFID                            = 207
	TIO_Ignition                        = 239
	TIO_MovementSensor                  = 240
	TIO_GSMOperator                     = 241
	TIO_Towing                          = 246
	TIO_CrashDetection                  = 247
	TIO_TripMode                        = 250
	TIO_ExcessiveIdling                 = 251
	TIO_Unplug                          = 252
	TIO_GreenDrivingStatus              = 253
	TIO_GreenDrivingValue               = 254
	TIO_Overspeeding                    = 255
	TIO_VIN                             = 256
	TIO_CrashTraceData                  = 257
	TIO_VIN_CAN                         = 325
	TIO_OdmTotalMileage                 = 389
	TIO_FuelLevel                       = 390
)

func (r *Record) ToProtobufDeviceStatus() *types.DeviceStatus {
	info := &types.DeviceStatus{}
	var speed float32

	info.Imei = r.IMEI
	info.DeviceType = types.DeviceType_TELTONIKA
	info.Timestamp = timestamppb.New(time.Unix(int64(r.Record.Timestamp), 0))

	// gps info
	info.Position = &types.GPSPosition{}
	info.Position.Latitude = r.Record.GPSElement.Latitude
	info.Position.Longitude = r.Record.GPSElement.Longitude
	info.Position.Altitude = float32(r.Record.GPSElement.Altitude)

	if r.Record.IOElement.Properties1B[TIO_Vehicle_Speed_OBD] > 0 {
		speed = float32(r.Record.IOElement.Properties1B[TIO_Vehicle_Speed_OBD])
	} else if r.Record.GPSElement.Speed > 0 {
		speed = float32(r.Record.GPSElement.Speed)
	} else if r.Record.IOElement.Properties1B[TIO_Speed_CAN] > 0 {
		speed = float32(r.Record.IOElement.Properties1B[TIO_Speed_CAN])
	}

	info.Position.Speed = &speed

	if r.Record.IOElement.Properties4B[TIO_OdmTotalMileage] > 0 {
		info.Odometer = int32(r.Record.IOElement.Properties4B[TIO_OdmTotalMileage])
	} else if r.Record.IOElement.Properties4B[TIO_TotalMileage_CAN] > 0 {

		info.Odometer = int32(r.Record.IOElement.Properties4B[TIO_TotalMileage_CAN]) / 1000
	} else if r.Record.IOElement.Properties4B[TIO_OdometerValue] > 0 {

		info.Odometer = int32(r.Record.IOElement.Properties4B[TIO_OdometerValue]) / 1000
	}

	info.Position.Course = float32(r.Record.GPSElement.Angle)
	info.Position.Satellites = int32(r.Record.GPSElement.Satellites)
	info.GsmNetwork = int32(r.Record.IOElement.Properties1B[TIO_GSMSignal])

	if r.Record.IOElement.Properties1B[TIO_AmbientTemperature] > 0 {
		info.Temperature = float32(r.Record.IOElement.Properties1B[TIO_AmbientTemperature] * 10)
	} else {
		info.Temperature = float32(r.Record.IOElement.Properties4B[TIO_DallasTemperature])
	}

	info.CoolantTemperature = float32(r.Record.IOElement.Properties1B[TIO_CoolantTemperature])

	info.EngineLoad = float32(r.Record.IOElement.Properties1B[TIO_EngineLoad])

	if r.Record.IOElement.Properties4B[TIO_FuelLevel] > 0 {
		info.FuelLtr = int32(r.Record.IOElement.Properties4B[TIO_FuelLevel]) / 10
	} else if r.Record.IOElement.Properties2B[TIO_FuelLevelLtr_CAN] > 0 {
		info.FuelLtr = int32(r.Record.IOElement.Properties2B[TIO_FuelLevelLtr_CAN]) / 10
	}

	if r.Record.IOElement.Properties1B[TIO_FuelLevelPCT_OBD] > 0 {
		info.FuelPct = int32(r.Record.IOElement.Properties1B[TIO_FuelLevelPCT_OBD])
	} else if r.Record.IOElement.Properties1B[TIO_FuelLevelPercent_CAN] > 0 {
		info.FuelPct = int32(r.Record.IOElement.Properties1B[TIO_FuelLevelPercent_CAN])
	}

	// Check if the iButtonID is available, otherwise use RFID
	if iButtonID, exists := r.Record.IOElement.Properties8B[TIO_IButtonID]; exists && iButtonID != 0 {
		info.IdentificationId = ConvertDecimalToHexAndReverse(iButtonID)
	} else if rfid, exists := r.Record.IOElement.Properties8B[TIO_RFID]; exists && rfid != 0 {
		info.IdentificationId = ConvertDecimalToHexAndReverse(rfid)
	} else {
		info.IdentificationId = ""
	}
	// vehicle status
	info.VehicleStatus = &types.VehicleStatus{}
	var ignition = r.Record.IOElement.Properties1B[TIO_DigitalInput1] > 0 || r.Record.IOElement.Properties1B[TIO_Ignition] > 0
	info.VehicleStatus.Ignition = &ignition
	info.VehicleStatus.AutoGeofence = r.Record.IOElement.Properties1B[TIO_AutoGeofence] > 0
	info.VehicleStatus.Towing = r.Record.IOElement.Properties1B[TIO_Towing] > 0
	info.VehicleStatus.UnplugBattery = r.Record.IOElement.Properties1B[TIO_Unplug] > 0
	info.VehicleStatus.OverSpeeding = r.Record.IOElement.Properties1B[TIO_Overspeeding] > 0
	info.VehicleStatus.RashDriving = r.Record.IOElement.Properties1B[TIO_GreenDrivingStatus] > 0
	info.VehicleStatus.CrashDetection = int32(r.Record.IOElement.Properties1B[TIO_CrashDetection]) > 0
	info.VehicleStatus.ExcessiveIdling = r.Record.IOElement.Properties1B[TIO_ExcessiveIdling] > 0

	if r.Record.IOElement.Properties2B[TIO_RPM] > 0 {
		info.Rpm = int32(r.Record.IOElement.Properties2B[TIO_RPM])
	} else if r.Record.IOElement.Properties2B[TIO_RPM_CAN] > 0 {
		info.Rpm = int32(r.Record.IOElement.Properties2B[TIO_RPM_CAN])
	}

	if string(r.Record.IOElement.PropertiesNXB[TIO_VIN]) != "" {
		info.Vin = string(r.Record.IOElement.PropertiesNXB[TIO_VIN])
	} else if string(r.Record.IOElement.PropertiesNXB[TIO_VIN_CAN]) != "" {
		info.Vin = string(r.Record.IOElement.PropertiesNXB[TIO_VIN_CAN])
	}

	info.FuelGps = int32(r.Record.IOElement.Properties4B[TIO_FuelUsedGPS] / 1000)

	//battery level
	if r.Record.IOElement.Properties2B[TIO_BatteryVoltage] <= 4200 {
		info.BatteryLevel = int32(r.Record.IOElement.Properties2B[TIO_BatteryVoltage] / 42)
	} else {
		info.BatteryLevel = int32(r.Record.IOElement.Properties2B[TIO_BatteryVoltage] / 90)
	}
	rawdata, _ := json.Marshal(r)
	info.RawData = &types.DeviceStatus_TeltonikaPacket{
		TeltonikaPacket: &types.TeltonikaPacket{RawData: rawdata},
	}
	return info
}

func (r *Response) ToProtobufDeviceResponse() *types.DeviceResponse {
	asciiMessage := string(r.Reply)

	// Log the entire decoded message as a single string, not character by character
	logger.Sugar().Infof("Full Response: %s", asciiMessage)

	return &types.DeviceResponse{
		Imei:     r.IMEI,
		Response: asciiMessage,
	}
}

// ConvertDecimalToHexAndReverse convert decimal to hex and then reverse the hex string
func ConvertDecimalToHexAndReverse(decimalValue uint64) string {
	// Step 0: Check if the decimal value is 0 (null equivalent for uint64)
	if decimalValue == 0 {
		return ""
	}

	// Step 1: Convert the decimal value to a hex string
	hexStr := fmt.Sprintf("%016X", decimalValue)

	// Step 2: Convert the hex string to bytes
	hexBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		fmt.Println("Error decoding hex string:", err)
		return ""
	}

	// Step 3: Reverse the byte order
	for i, j := 0, len(hexBytes)-1; i < j; i, j = i+1, j-1 {
		hexBytes[i], hexBytes[j] = hexBytes[j], hexBytes[i]
	}

	// Step 4: Convert the reversed bytes back to a hex string
	reversedHexStr := hex.EncodeToString(hexBytes)

	return reversedHexStr
}
