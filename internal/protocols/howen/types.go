package howen

// Import necessary packages
import (
	"encoding/json"
	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strconv"
	"time"
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

type Storage struct {
	Name   string `json:"name"`   // Name of the storage device (e.g., "sd1")
	Index  string `json:"index"`  // Index of the storage device (e.g., "0")
	Status string `json:"status"` // Status of the storage device (e.g., "1" for active)
	Total  string `json:"total"`  // Total capacity of the storage in MB or GB (e.g., "7523")
	Free   string `json:"free"`   // Free space available in MB or GB (e.g., "0")
}

// Payload represents the main payload structure, holding different device details.
type Payload struct {
	Alarm      Alarm             `json:"alarm"`
	Basic      Basic             `json:"basic"`
	Bluetooth  Bluetooth         `json:"bluetooth"`
	DeviceID   string            `json:"deviceID"`
	DeviceTemp DeviceTemp        `json:"deviceTemp"`
	Driver     Driver            `json:"driver"`
	DTU        string            `json:"dtu"`
	Ext        Extra             `json:"ext"`
	Fuel       map[string]string `json:"fuel"`
	GSensor    GSensor           `json:"gsensor"`
	Load       Load              `json:"load"`
	Location   Location          `json:"location"`
	Module     Module            `json:"module"`
	Mileage    Mileage           `json:"mileage"`
	Mobile     map[string]string `json:"mobile"`
	NodeID     string            `json:"nodeID"`
	OBD        []OBD             `json:"obd,omitempty"` // Array of OBD data
	Storage    []Storage         `json:"storage"`
	Temp       Temp              `json:"temp"`
	Voltage    Voltage           `json:"voltage"`
	Wifi       map[string]string `json:"wifi"`
}

type OBD struct {
	TotalMil      int     `json:"totalMil"`      // Total mileage
	TotalFuel     int     `json:"totalFuel"`     // Total fuel consumed
	InstanFuel    float64 `json:"instanFuel"`    // Instantaneous fuel consumption
	Voltage       float64 `json:"voltage"`       // Vehicle voltage
	RPM           int     `json:"rpm"`           // Engine revolutions per minute
	Speed         float64 `json:"speed"`         // Vehicle speed in km/h
	AirShed       float64 `json:"airshed"`       // Air intake flow rate
	StressMpa     float64 `json:"stressMpa"`     // Air intake pressure in kPa
	CoolantsTemp  int     `json:"coolantsTemp"`  // Coolant temperature
	AirTemp       int     `json:"airTemp"`       // Air intake temperature
	MotorLimit    int     `json:"motorLimit"`    // Engine load in percentage
	Position      int     `json:"position"`      // Throttle position in percentage
	EFOA          int     `json:"efoa"`          // Fuel tank level in percentage
	VIN           string  `json:"vin"`           // Vehicle Identification Number
	Engine        int     `json:"engine"`        // Engine status (1: ON, 0: OFF)
	Idle          int     `json:"idle"`          // Idle status (1: Start, 0: End)
	EngineOnTime  string  `json:"engineOnTime"`  // Engine ON time
	EngineOffTime string  `json:"engineOffTime"` // Engine OFF time
	HC            int     `json:"hc"`            // Harsh cornering events
	HA            int     `json:"ha"`            // Harsh acceleration events
	HB            int     `json:"hb"`            // Harsh braking events
	LowBV         int     `json:"lowbv"`         // Battery low voltage (0: No, 1: Yes)
}

// AlarmPayload represents payload data for alarms.
type AlarmPayload struct {
	Alarm       Alarm             `json:"alarm"`
	AlarmDetail string            `json:"alarmDetail"`
	AlarmID     string            `json:"alarmID"`
	Basic       Basic             `json:"basic"`
	Bluetooth   Bluetooth         `json:"bluetooth"`
	DeviceID    string            `json:"deviceID"`
	DeviceTemp  DeviceTemp        `json:"deviceTemp"`
	Driver      Driver            `json:"driver"`
	DTU         string            `json:"dtu"`
	EventType   string            `json:"eventType"`
	Fuel        map[string]string `json:"fuel"`
	GSensor     GSensor           `json:"gsensor"`
	IsLater     int               `json:"isLater"`
	Location    Location          `json:"location"`
	Load        Load              `json:"load"`
	Mileage     Mileage           `json:"mileage"`
	Module      Module            `json:"module"`
	Mobile      map[string]string `json:"mobile"`
	NodeID      string            `json:"nodeID"`
	OBD         []OBD             `json:"obd,omitempty"` // Array of OBD data
	Payload     PayloadDetail     `json:"payload"`
	Storage     []Storage         `json:"storage"`
	Temp        Temp              `json:"temp"`
	Voltage     Voltage           `json:"voltage"`

	Wifi map[string]string `json:"wifi"`
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

type AlarmDetails struct {
	AlarmID   string    `json:"alarmID"`  // Alarm unique identifier
	DeviceID  string    `json:"deviceID"` // Device identifier
	StartTime time.Time `json:"st"`       // Alarm start time
	EndTime   time.Time `json:"et"`       // Alarm end time
	Details   string    `json:"det"`      // Detailed alarm information
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
	speed := parseToFloat32(p.Payload.Location.Speed)
	info.Position = &types.GPSPosition{
		Latitude:   parseToFloat32(p.Payload.Location.Latitude),
		Longitude:  parseToFloat32(p.Payload.Location.Longitude),
		Altitude:   parseToFloat32(p.Payload.Location.Altitude),
		Speed:      &speed,
		Satellites: parseToInt32(p.Payload.Location.Satellites),
		Course:     parseToFloat32(p.Payload.Location.Direct),
	}

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
	info.Timestamp = timestamppb.Now() // You may replace this with the event's actual timestamp if available

	// Device-specific raw data
	rawdata, _ := json.Marshal(a)
	info.RawData = &types.DeviceStatus_HowenPacket{
		HowenPacket: &types.HowenPacket{RawData: rawdata},
	}

	speed := parseToFloat32(a.Payload.Location.Speed)
	loc := a.Payload.Location
	// Populate GPS position if available
	if parseToFloat32(loc.Latitude) != 0 || parseToFloat32(loc.Longitude) != 0 {
		info.Position = &types.GPSPosition{
			Latitude:  parseToFloat32(loc.Latitude),
			Longitude: parseToFloat32(loc.Longitude),
			Altitude:  parseToFloat32(loc.Altitude),
			Speed:     &speed,
			Course:    parseToFloat32(loc.Direct),
		}
	}

	// Initialize vehicle status
	vehicleStatus := &types.VehicleStatus{}
	vehicleStatus.Ignition = parseIgnition(parseToFloat32(a.Payload.Basic.Key))
	// Handle event codes (EC)
	switch a.Payload.Payload.EC {
	case 217: // Car Crash , impact
		vehicleStatus.CrashDetection = true

	//case XXX: // Deviation from Route
	//	info.MessageType = "Deviation from Route"

	case 103: // Distance Between Objects
		//info.VehicleStatus.DistanceBetweenObjects

	case 126: // Driver Absence
		//info.VehicleStatus.DriverAbsence

	//case XXX: // Driver Change
	//	info.MessageType = "Driver Change"

	case 118: // Driver Distraction
		vehicleStatus.DriverDistraction = true

	case 32: // Engine Excessive Idling
		vehicleStatus.ExcessiveIdling = true

	case 200: // Entrance  Geofence
		vehicleStatus.EntringGeofence = true
	case 201: // Exiting  Geofence
		vehicleStatus.ExitingGeofence = true

	case 17: // Excessive Driving, fatigue Driving
		vehicleStatus.FatigueDriving = true

	case 11: // Excessive Parking
		vehicleStatus.ExcessiveParking = true

	//case 17: // Fatigue Driving
	//	info.MessageType = "Fatigue Driving"

	case 210: // Fuel Level Change (Refuel)
		vehicleStatus.FuelRefuel = true

	case 211: // Fuel Level Change (Fuel Theft)
		vehicleStatus.FuelTheft = true

	case 24, 25, 26, 27, 214, 212, 213: // Harsh Driving
		vehicleStatus.RashDriving = true

	case 220, 221, 222, 223, 224, 225, 226, 227: // Inputs Triggering
		vehicleStatus.InputsTriggering = true

	case 228, 229: // Outputs Triggering
		vehicleStatus.OutputsTriggering = true

	//case XXX: // Parameter in Range
	//	info.MessageType = "Parameter in Range"

	//case XXX: // Parking State Detection
	//	info.MessageType = "Parking State Detected"

	//case XXX: // Pressing SOS Button
	//	info.MessageType = "SOS Button Pressed"

	case 48: // Speeding

		vehicleStatus.OverSpeeding = true

	//case XXX: // State Field Value
	//	info.MessageType = "State Field Value"

	//case XXX: // Task Status Change
	//	info.MessageType = "Task Status Changed"

	case 1: // Tracker Switched OFF or Lost Connection
		vehicleStatus.TrackerOffline = true

	case 218: // Vibration Sensor (Rollover)

		vehicleStatus.CrashDetection = true

	default: // Unknown or unsupported event
		info.MessageType = "Unknown Event"
	}

	// Attach vehicle status if populated
	if vehicleStatus.OverSpeeding || vehicleStatus.RashDriving || vehicleStatus.CrashDetection || vehicleStatus.ExcessiveIdling {
		info.VehicleStatus = vehicleStatus
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
