package howen

// Import necessary packages
import (
	"encoding/json"
	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strconv"
	_ "time"
)

// Actionrepresents different types of actions, such as login, subscription, GPS data, alarm data, and status.
type Action struct {
	Type      string         `json:"type"`                // e.g., "80000", "80001", "80003", etc.
	Login     *LoginData     `json:"login,omitempty"`     // For action "80000"
	Subscribe *SubscribeData `json:"subscribe,omitempty"` // For action "80001"
	GPS       *GPSData       `json:"gps,omitempty"`       // For action "80003"
	Alarm     *AlarmData     `json:"alarm,omitempty"`     // For action "80004"
	Status    *StatusData    `json:"status,omitempty"`    // For action "80005"
}

type ActionData struct {
	Action  string  `json:"action"`
	Payload Payload `json:"payload"`
}

// LoginData holds login information.
type LoginData struct {
	Username string `json:"username"`
	Token    string `json:"token"`
	PID      string `json:"pid"`
}

// SubscribeData represents data for subscription actions.
type SubscribeData struct {
	Channel string `json:"channel"`
}

// GPSData holds GPS information.
type GPSData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Speed     float64 `json:"speed"`
	Altitude  float64 `json:"altitude"`
}

// AlarmData represents information about alarms.
type AlarmData struct {
	AlarmType string `json:"alarm_type"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
}

// StatusData indicates the online/offline status and the time of status change.
type StatusData struct {
	Online bool   `json:"online"`
	Time   string `json:"time"` // Timestamp of status change
}

// LoginResponse represents the response to a login action.
type LoginResponse struct {
	Msg    string `json:"msg"`
	Result string `json:"result"`
}

// DevicePayload holds details of a device's specifications and metadata.
type DevicePayload struct {
	GMT      string `json:"gmt"`
	Firmware string `json:"fw"`
	Ext      string `json:"ext"`
	MCU      string `json:"mcu"`
	HW       string `json:"hw"`
	IMEI     string `json:"imei"`
	UM       string `json:"um"`
	Dial     string `json:"dial"`
	ALG      string `json:"alg"`
}

// Location represents location-related data.
type Location struct {
	Mode       string `json:"mode"`
	DTU        string `json:"dtu"`
	Direct     string `json:"direct"`
	Satellites string `json:"satellites"`
	Speed      string `json:"speed"`
	Altitude   string `json:"altitude"`
	Precision  string `json:"precision"`
	Longitude  string `json:"longitude"`
	Latitude   string `json:"latitude"`
}

// GSensor holds G-sensor data.
type GSensor struct {
	X    string `json:"x"`
	Y    string `json:"y"`
	Z    string `json:"z"`
	Tilt string `json:"tilt"`
	Hit  string `json:"hit"`
}

// Module represents various module configurations.
type Module struct {
	Mobile   string `json:"mobile"`
	Location string `json:"location"`
	Wifi     string `json:"wifi"`
	GSensor  string `json:"gsensor"`
	Record   string `json:"record"`
}

// Alarm holds alarm-related information.
type Alarm struct {
	VideoLost     string `json:"videoLost"`
	MotionDection string `json:"motionDection"`
	VideoMask     string `json:"videoMask"`
	Input         string `json:"input"`
	OverSpeed     string `json:"overSpeed"`
	LowSpeed      string `json:"lowSpeed"`
	Urgency       string `json:"urgency"`
}

// Temp represents temperature data.
type Temp struct {
	InsideTemp      string `json:"insideTemp"`
	OutsideTemp     string `json:"outsideTemp"`
	EngineerTemp    string `json:"engineerTemp"`
	DeviceTemp      string `json:"deviceTemp"`
	InsideHumidity  string `json:"insideHumidity"`
	OutsideHumidity string `json:"outsideHumidity"`
}

// Mileage represents mileage information.
type Mileage struct {
	TodayDay string `json:"todayDay"`
	Total    string `json:"total"`
}

// Voltage holds voltage data.
type Voltage struct {
	VCC string `json:"vcc"`
	Bat string `json:"bat"`
}

// Driver represents driver information.
type Driver struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Bluetooth represents Bluetooth connection status.
type Bluetooth struct {
	Connect string `json:"connect"`
}

// Load represents load status.
type Load struct {
	Status string `json:"status"`
}

// DeviceTemp holds information about device temperatures.
type DeviceTemp struct {
	CPU  string `json:"cpu"`
	Disk string `json:"disk"`
}

// Basic represents a basic key-value structure.
type Basic struct {
	Key string `json:"key"`
}

// Payload represents the main payload structure, holding different device details.
type Payload struct {
	DeviceID   string            `json:"deviceID"`
	NodeID     string            `json:"nodeID"`
	DTU        string            `json:"dtu"`
	Location   Location          `json:"location"`
	GSensor    GSensor           `json:"gsensor"`
	Basic      Basic             `json:"basic"`
	Module     Module            `json:"module"`
	Fuel       map[string]string `json:"fuel"`
	Mobile     map[string]string `json:"mobile"`
	Wifi       map[string]string `json:"wifi"`
	Storage    string            `json:"storage"`
	Alarm      Alarm             `json:"alarm"`
	Temp       Temp              `json:"temp"`
	Mileage    Mileage           `json:"mileage"`
	Voltage    Voltage           `json:"voltage"`
	Driver     Driver            `json:"driver"`
	Bluetooth  Bluetooth         `json:"bluetooth"`
	Load       Load              `json:"load"`
	DeviceTemp DeviceTemp        `json:"deviceTemp"`
	Ext        Extra             `json:"ext"`
}

// AlarmPayload represents payload data for alarms.
type AlarmPayload struct {
	GSensor     GSensor           `json:"gsensor"`
	Fuel        map[string]string `json:"fuel"`
	IsLater     int               `json:"isLater"`
	Storage     string            `json:"storage"`
	Load        Load              `json:"load"`
	DeviceTemp  DeviceTemp        `json:"deviceTemp"`
	Payload     PayloadDetail     `json:"payload"`
	Alarm       Alarm             `json:"alarm"`
	NodeID      string            `json:"nodeID"`
	Mileage     Mileage           `json:"mileage"`
	Wifi        map[string]string `json:"wifi"`
	Temp        Temp              `json:"temp"`
	DTU         string            `json:"dtu"`
	Module      Module            `json:"module"`
	Mobile      map[string]string `json:"mobile"`
	EventType   string            `json:"eventType"`
	DeviceID    string            `json:"deviceID"`
	Bluetooth   Bluetooth         `json:"bluetooth"`
	Voltage     Voltage           `json:"voltage"`
	AlarmDetail string            `json:"alarmDetail"`
	Driver      Driver            `json:"driver"`
	AlarmID     string            `json:"alarmID"`
	Location    Location          `json:"location"`
	Basic       Basic             `json:"basic"`
}

// AlarmMessage represents an alarm message.
type AlarmMessage struct {
	Action  string       `json:"action"`
	Payload AlarmPayload `json:"payload"`
}

// GPSPacket represents a GPS data packet.
type GPSPacket struct {
	Action  string  `json:"action"`
	Payload Payload `json:"payload"`
}

// DeviceStatus represents the status of a device.
type DeviceStatus struct {
	Action  string        `json:"action"`
	Payload DevicePayload `json:"payload"`
}

// PayloadDetail holds detailed information within a payload.
type PayloadDetail struct {
	ST     string `json:"st"`
	Det    Det    `json:"det"`
	DTU    string `json:"dtu"`
	DrID   string `json:"drid"`
	DrName string `json:"drname"`
	SPDS   string `json:"spds"`
	UUID   string `json:"uuid"`
	EC     int    `json:"ec"`
	ET     string `json:"et"`
}

// Det represents detailed information within PayloadDetail.
type Det struct {
	DT  string `json:"dt"`
	Cur string `json:"cur"`
	VT  string `json:"vt"`
}

// Extra holds additional metadata.
type Extra struct {
	FW string `json:"fw"`
	// Add any other fields that `Extra` should contain
}

func (p *GPSPacket) ToProtobufDeviceStatusGPS() *types.DeviceStatus {
	info := &types.DeviceStatus{}

	// Fill common device information
	info.Imei = p.Payload.DeviceID
	info.DeviceType = types.DeviceType_HOWEN
	info.Timestamp = timestamppb.Now() // Or use a specific timestamp from p.Payload

	// Populate GPS-specific data
	info.Position = &types.GPSPosition{
		Latitude:  parseToFloat32(p.Payload.Location.Latitude),
		Longitude: parseToFloat32(p.Payload.Location.Longitude),
		Altitude:  parseToFloat32(p.Payload.Location.Altitude),
	}
	speed := parseToFloat32(p.Payload.Location.Speed)
	info.Position.Speed = &speed
	info.Position.Satellites = parseToInt32(p.Payload.Location.Satellites)

	// Populate additional fields (adjust these based on Howen-specific data mappings)
	info.BatteryLevel = int32(parseToFloat32(p.Payload.Voltage.Bat) * 100 / 12)
	info.FuelLtr = int32(parseToFloat32(p.Payload.Fuel["total"]))
	info.Rpm = int32(parseToFloat32(p.Payload.Module.Mobile))
	info.Odometer = int32(parseToFloat32(p.Payload.Mileage.Total))

	info.VehicleStatus = &types.VehicleStatus{
		Ignition: parseIgnition(parseToFloat32(p.Payload.Basic.Key)),
	}

	// Device-specific raw data
	rawdata, _ := json.Marshal(p)
	info.RawData = &types.DeviceStatus_HowenPacket{
		HowenPacket: &types.HowenPacket{RawData: rawdata},
	}

	return info
}

// Convert AlarmMessage data to DeviceStatus protobuf struct.

func (a *AlarmMessage) ToProtobufDeviceStatusAlarm() *types.DeviceStatus {
	info := &types.DeviceStatus{}

	// Fill common device information
	info.Imei = a.Payload.DeviceID
	info.DeviceType = types.DeviceType_HOWEN
	info.Timestamp = timestamppb.Now() // Or use a specific timestamp from a.Payload

	// Populate alarm-specific fields

	// Device-specific raw data
	rawdata, _ := json.Marshal(a)
	info.RawData = &types.DeviceStatus_HowenPacket{
		HowenPacket: &types.HowenPacket{RawData: rawdata},
	}

	return info
}

func parseToFloat32(value string) float32 {
	if value == "" {
		return 0
	}
	f, err := strconv.ParseFloat(value, 32)
	if err != nil {
		logger.Sugar().Info("error parsing float32: %v\n", err)
		return 0
	}
	return float32(f)
}

func parseToInt32(value string) int32 {
	i, err := strconv.Atoi(value)
	if err != nil {
		logger.Sugar().Info("error parsing int32: %v\n", err)
		return 0
	}
	return int32(i)
}

func parseIgnition(value float32) *bool {
	ignition := value == 1.0 // true if value is 1.0, false otherwise
	return &ignition
}
