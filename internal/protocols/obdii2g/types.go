package obdii2g

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	packetHeader         = "$$CLIENT_1NS"
	obdDataSeparator     = "|"
	checksumSeparator    = "*"
	timeFormat           = "060102150405"
	maxPacketSize        = 2048
	defaultBatteryCutoff = 3000 // 3V minimum for Li-ion
)

var (
	ErrInvalidPacket      = fmt.Errorf("invalid packet format")
	ErrChecksumMismatch   = fmt.Errorf("checksum mismatch")
	ErrInvalidTimestamp   = fmt.Errorf("invalid timestamp")
	ErrInvalidCoordinates = fmt.Errorf("invalid coordinates")
)

type Packet struct {
	Raw         string
	IMEI        string
	MessageCode string
	Timestamp   time.Time
	Position    GPSCoordinates
	Vehicle     VehicleData
	OBD         map[string]OBDParameter
	Checksum    byte
	IsValid     bool
}

type GPSCoordinates struct {
	Latitude   float64
	Longitude  float64
	Speed      float64
	Course     float64
	Satellites int32
	HDOP       float64
	FixStatus  bool
}

type VehicleData struct {
	EventFlag       uint32
	GSMSignal       int32
	AnalogInput     int32
	AccumulatedDist int32
	ExternalBattery int32
	InternalBattery int32
	TripTime        int32
}

type OBDParameter struct {
	RawHex string
	Parsed float64
	Unit   string
	Valid  bool
}

const (
	pidEngineRpm            = "010C" // 2 bytes: (A*256 + B) / 4 rpm
	pidVehicleSpeed         = "010D" // 1 byte : A km/h
	pidEngineLoad           = "0104" // 1 byte : A *100/255 %
	pidCoolantTemp          = "0105" // 1 byte : A - 40 °C
	pidIntakeAirTemp        = "010F" // 1 byte : A - 40 °C
	pidRunTime              = "011F" // 2 bytes: A*256 + B s
	pidDistanceMILOn        = "0121" // 2 bytes: A*256 + B km
	pidFuelLevelInput       = "012F" // 1 byte : A *100/255 %
	pidMafAirFlow           = "0110" // 2 bytes: (A*256 + B)/100 g/s
	pidFuelRailPressure     = "0122" // 2 bytes: (A*256 + B)*0.079 kPa
	pidWarmupsSinceClear    = "0130" // 1 byte : A count
	pidDistanceSinceClear   = "0131" // 2 bytes: A*256 + B km
	pidBarometricPressure   = "0133" // 1 byte : A kPa
	pidControlModuleVoltage = "0142" // 2 bytes: (A*256 + B)/1000 V
	pidAbsoluteLoadValue    = "0143" // 2 bytes: (A*256 + B)*100/255 %
	pidEngineOilTemp        = "015C" // 1 byte : A - 40 °C
)

// ToProtobuf converts raw packet to protobuf format with complete data mapping
func (p *Packet) ToProtobuf() (*types.DeviceStatus, error) {
	if !p.IsValid {
		return nil, fmt.Errorf("cannot convert invalid packet")
	}
	speed := float32(p.Position.Speed)
	if obdSpeed := p.getOBDFloatValue(pidVehicleSpeed); obdSpeed != 0 {
		speed = obdSpeed
	}

	logger.Sugar().Infow("obdii2g_packet",
		// envelope
		"imei", p.IMEI,
		"message_code", p.MessageCode,
		"received_at", time.Now().Format(time.RFC3339),

		// timestamp & GPS
		"device_timestamp", p.Timestamp.Format(time.RFC3339),
		"latitude", p.Position.Latitude,
		"longitude", p.Position.Longitude,
		"speed_gps_kmh", p.Position.Speed,
		"course_deg", p.Position.Course,
		"hdop", p.Position.HDOP,
		"satellites", p.Position.Satellites,
		"fix_ok", p.Position.FixStatus,

		// vehicle
		"gsm_signal", p.Vehicle.GSMSignal,
		"odometer_km", p.Vehicle.AccumulatedDist,
		"analog_input_mv", p.Vehicle.AnalogInput,
		"event_flags", fmt.Sprintf("0x%08X", p.Vehicle.EventFlag),
		"external_batt_mv", p.Vehicle.ExternalBattery,
		"internal_batt_mv", p.Vehicle.InternalBattery,
		"trip_time_s", p.Vehicle.TripTime,

		// OBD-II PIDs (only valid ones will be non-zero)
		"engine_rpm", p.getOBDFloatValue(pidEngineRpm),
		"vehicle_speed_kmh", p.getOBDFloatValue(pidVehicleSpeed),
		"engine_load_pct", p.getOBDFloatValue(pidEngineLoad),
		"intake_air_temp_c", p.getOBDFloatValue(pidIntakeAirTemp),
		"coolant_temp_c", p.getOBDFloatValue(pidCoolantTemp),
		"engine_oil_temp_c", p.getOBDFloatValue(pidEngineOilTemp),
		"run_time_s", p.getOBDFloatValue(pidRunTime),
		"dist_mil_on_km", p.getOBDFloatValue(pidDistanceMILOn),
		"warmups_since_clear", p.getOBDFloatValue(pidWarmupsSinceClear),
		"dist_since_clear_km", p.getOBDFloatValue(pidDistanceSinceClear),
		"fuel_level_pct", p.getOBDFloatValue(pidFuelLevelInput),
		"baro_pressure_kpa", p.getOBDFloatValue(pidBarometricPressure),
		"maf_airflow_gps", p.getOBDFloatValue(pidMafAirFlow),
		"fuel_rail_press_kpa", p.getOBDFloatValue(pidFuelRailPressure),
		"module_voltage_v", p.getOBDFloatValue(pidControlModuleVoltage),
		"abs_load_value_pct", p.getOBDFloatValue(pidAbsoluteLoadValue),
	)

	proto := &types.DeviceStatus{
		Imei:       p.IMEI,
		DeviceType: types.DeviceType_AQUILA,
		Timestamp:  timestamppb.New(p.Timestamp),
		Position: &types.GPSPosition{
			Latitude:   float32(p.Position.Latitude),
			Longitude:  float32(p.Position.Longitude),
			Speed:      &speed,
			Course:     float32(p.Position.Course),
			Satellites: p.Position.Satellites,
		},
		VehicleStatus:        p.parseVehicleStatus(),
		BatteryLevel:         p.calculateBatteryLevel(),
		Odometer:             p.Vehicle.AccumulatedDist,
		GsmNetwork:           convertGSMSignalToLevel(p.Vehicle.GSMSignal),
		FuelPct:              p.getFuelData(),
		Rpm:                  int32(p.getOBDFloatValue(pidEngineRpm)),
		CoolantTemperature:   p.getOBDFloatValue(pidCoolantTemp),
		EngineLoad:           p.getOBDFloatValue(pidEngineLoad),
		IntakeAirTemp:        p.getOBDFloatValue(pidIntakeAirTemp),
		RunTime:              uint32(p.getOBDFloatValue(pidRunTime)),
		DistanceMilOn:        uint32(p.getOBDFloatValue(pidDistanceMILOn)),
		FuelLevelInput:       p.getOBDFloatValue(pidFuelLevelInput),
		MafAirFlow:           p.getOBDFloatValue(pidMafAirFlow),
		FuelRailPressure:     p.getOBDFloatValue(pidFuelRailPressure),
		WarmupsSinceClear:    uint32(p.getOBDFloatValue(pidWarmupsSinceClear)),
		DistanceSinceClear:   uint32(p.getOBDFloatValue(pidDistanceSinceClear)),
		BarometricPressure:   int32(p.getOBDFloatValue(pidBarometricPressure)),
		ControlModuleVoltage: p.getOBDFloatValue(pidControlModuleVoltage),
		AbsoluteLoadValue:    p.getOBDFloatValue(pidAbsoluteLoadValue),
		EngineOilTemp:        p.getOBDFloatValue(pidEngineOilTemp),
		RawData: &types.DeviceStatus_AquilaPacket{
			AquilaPacket: &types.AquilaPacket{
				RawData: []byte(p.Raw),
			},
		},
	}

	return proto, nil
}

// parseVehicleStatus decodes 32-bit event flag to 25+ status fields
func (p *Packet) parseVehicleStatus() *types.VehicleStatus {
	vs := &types.VehicleStatus{}
	flags := p.Vehicle.EventFlag
	logger.Sugar().Infoln("flags", flags)
	// vs.HarshBrakingEvent      = (flags & (1 << 26)) != 0 // bit 27
	vs.UnplugBattery = (flags & (1 << 2)) != 0
	vs.OverSpeeding = (flags & (1 << 3)) != 0

	if ig := (flags & (1 << 11)) != 0; ig {
		vs.Ignition = &ig
	}

	vs.Towing = (flags & (1 << 13)) != 0
	vs.CrashDetection = (flags & (1 << 25)) != 0
	vs.RashDriving = (flags & (1 << 26)) != 0
	// vs.HarshAcceleration = (flags & (1 << 26)) != 0
	vs.HarshBraking = (flags & (1 << 27)) != 0

	return vs
}

// func (p *Packet) parseBitFlag(binaryStr string, bitPos int) bool {
// 	if len(binaryStr) != 32 || bitPos < 0 || bitPos > 31 {
// 		return false
// 	}
// 	return binaryStr[bitPos] == '1'
// }

func (p *Packet) calculateBatteryLevel() int32 {
	if p.Vehicle.ExternalBattery == 0 {
		return 0
	}

	// Li-ion battery calculation (3.0V - 4.2V range)
	voltage := float32(p.Vehicle.ExternalBattery) / 1000
	percentage := (voltage - 3.0) / (4.2 - 3.0) * 100
	return int32(clamp(percentage, 0, 100))
}

func (p *Packet) getFuelData() int32 {
	// PID 012F: Fuel Level Input
	if fuel, exists := p.OBD[pidFuelLevelInput]; exists && fuel.Valid {
		return int32(fuel.Parsed)
	}
	return 0
}

func (p *Packet) getOBDFloatValue(pid string) float32 {
	if param, exists := p.OBD[pid]; exists && param.Valid {
		return float32(param.Parsed)
	}
	return 0
}

// ParsePacket validates and parses complete Aquila packet
func ParsePacket(raw string) (*Packet, error) {
	// if !strings.HasPrefix(raw, packetHeader) {
	// 	return nil, fmt.Errorf("%w: invalid header", ErrInvalidPacket)
	// }
	// if len(raw) > maxPacketSize {
	// 	return nil, fmt.Errorf("%w: packet size exceeded", ErrInvalidPacket)
	// }

	// // Split off checksum
	// parts := strings.Split(raw, *)
	// if len(parts) != 2 || len(parts[1]) != 2 {
	// 	return nil, fmt.Errorf("%w: missing checksum", ErrInvalidPacket)
	// }
	// calculated := calculateChecksum(parts[0])
	// received, err := hex.DecodeString(parts[1])
	// if err != nil || calculated != received[0] {
	// 	return nil, fmt.Errorf("%w: expected %02X got %s",
	// 		ErrChecksumMismatch, calculated, parts[1])
	// }

	// Split the core comma‐fields
	fields := strings.Split(raw, ",")
	if len(fields) < 22 {
		return nil, fmt.Errorf("%w: insufficient fields", ErrInvalidPacket)
	}

	pkt := &Packet{
		Raw:     raw,
		IsValid: true,
	}

	// ── Core fields ───────────────────────────────────────
	pkt.IMEI = fields[1] // was fields[2]
	pkt.MessageCode = fields[2]
	// Timestamp is at index 5 (YYMMDDhhmmss)
	if ts, err := time.Parse(timeFormat, fields[5]); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTimestamp, err)
	} else {
		pkt.Timestamp = ts
	}

	// ── GPS ────────────────────────────────────────────────
	pkt.Position.Latitude, _ = parseFloat(fields[3])
	pkt.Position.Longitude, _ = parseFloat(fields[4])
	pkt.Position.FixStatus = (fields[6] == "A") // was fields[7]
	pkt.Position.Satellites, _ = parseInt32(fields[11])
	pkt.Position.Course, _ = parseFloat(fields[10]) // was [11]
	pkt.Position.Speed, _ = parseFloat(fields[8])   // was [9]
	pkt.Position.HDOP, _ = parseFloat(fields[12])   // was [13]

	// ── Vehicle ──────────────────────────────────────────
	pkt.Vehicle.GSMSignal, _ = parseInt32(fields[7])
	pkt.Vehicle.AccumulatedDist, _ = parseInt32(fields[9]) // was [10]
	pkt.Vehicle.AnalogInput, _ = parseInt32(fields[15])
	pkt.Vehicle.EventFlag, _ = parseUint32(fields[16])
	pkt.Vehicle.ExternalBattery, _ = parseInt32(fields[17])
	pkt.Vehicle.InternalBattery, _ = parseInt32(fields[18])
	pkt.Vehicle.TripTime, _ = parseInt32(fields[19])
	logger.Sugar().Infoln("fields[20]", fields[20])
	// ── OBD Parameters ────────────────────────────────────
	pkt.OBD = make(map[string]OBDParameter)
	// The last CSV field before '*' is something like "1|PID:HEX|PID:HEX|…"
	obdField := fields[len(fields)-1]
	entries := strings.Split(obdField, obdDataSeparator)
	if len(entries) > 1 {
		// skip the leading count ("1")
		for _, entry := range entries[1:] {
			pidParts := strings.SplitN(entry, ":", 2)
			if len(pidParts) != 2 {
				continue
			}
			param := OBDParameter{RawHex: pidParts[1]}
			param.parseValue(pidParts[0])
			pkt.OBD[pidParts[0]] = param
		}
	}

	return pkt, nil
}

func (o *OBDParameter) parseValue(pidCode string) {
	defer func() {
		if r := recover(); r != nil {
			o.Valid = false
		}
	}()

	// 1) decode the full hex string into bytes
	rawBytes, err := hex.DecodeString(strings.TrimSpace(o.RawHex))
	if err != nil || len(rawBytes) < 3 {
		o.Valid = false
		return
	}
	// rawBytes[0] = Length
	// rawBytes[1] = ResponseType (0x40 + Mode for positive, or 0x7F for negative)
	// rawBytes[2] = echoed PID
	// the real data starts at rawBytes[3:]
	if rawBytes[1] == 0x7F {
		// negative response: NACK
		o.Valid = false
		return
	}

	data := rawBytes[3:] // <-- now data[0], data[1], ... are the actual payload bytes

	switch pidCode {
	case pidEngineRpm:
		if len(data) < 2 {
			o.Valid = false
			return
		}
		raw := uint16(data[0])<<8 | uint16(data[1])
		o.Parsed = float64(raw) / 4
		o.Unit = "rpm"

	case pidVehicleSpeed:
		if len(data) < 1 {
			o.Valid = false
			return
		}
		o.Parsed = float64(data[0])
		o.Unit = "km/h"

	case pidEngineLoad:
		if len(data) < 1 {
			o.Valid = false
			return
		}
		o.Parsed = float64(data[0]) * 100 / 255
		o.Unit = "%"

	case pidCoolantTemp, pidIntakeAirTemp, pidEngineOilTemp:
		if len(data) < 1 {
			o.Valid = false
			return
		}
		o.Parsed = float64(data[0]) - 40
		o.Unit = "°C"

	case pidRunTime, pidDistanceMILOn, pidDistanceSinceClear, pidMafAirFlow,
		pidFuelRailPressure, pidControlModuleVoltage, pidAbsoluteLoadValue:
		// all these need at least two bytes
		if len(data) < 2 {
			o.Valid = false
			return
		}
		// fall through to specialized logic below
		switch pidCode {
		case pidRunTime:
			raw := uint16(data[0])<<8 | uint16(data[1])
			o.Parsed = float64(raw)
			o.Unit = "s"

		case pidDistanceMILOn, pidDistanceSinceClear:
			raw := uint16(data[0])<<8 | uint16(data[1])
			o.Parsed = float64(raw)
			o.Unit = "km"

		case pidMafAirFlow:
			raw := uint16(data[0])<<8 | uint16(data[1])
			o.Parsed = float64(raw) / 100
			o.Unit = "g/s"

		case pidFuelRailPressure:
			raw := uint16(data[0])<<8 | uint16(data[1])
			o.Parsed = float64(raw) * 0.079
			o.Unit = "kPa"

		case pidControlModuleVoltage:
			raw := uint16(data[0])<<8 | uint16(data[1])
			o.Parsed = float64(raw) / 1000
			o.Unit = "V"

		case pidAbsoluteLoadValue:
			raw := uint16(data[0])<<8 | uint16(data[1])
			o.Parsed = float64(raw) * 100 / 255
			o.Unit = "%"

		}
	case pidFuelLevelInput, pidWarmupsSinceClear, pidBarometricPressure:
		switch pidCode {
		case pidFuelLevelInput:
			o.Parsed = float64(data[0]) * 100 / 255
			o.Unit = "%"
		case pidWarmupsSinceClear:
			o.Parsed = float64(data[0])
			o.Unit = "cycles"
		case pidBarometricPressure:
			o.Parsed = float64(data[0])
			o.Unit = "kPa"
		}
	default:
		// unsupported PID
		o.Valid = false
		return
	}

	o.Valid = true
}

// Helper functions
func calculateChecksum(data string) byte {
	var cs byte
	for _, c := range []byte(data) {
		cs ^= c
	}
	return cs
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

func parseInt32(s string) (int32, error) {
	i, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	return int32(i), err
}

func parseUint32(s string) (uint32, error) {
	i, err := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
	return uint32(i), err
}

func clamp(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// convert raw 0–31 (+99 unknown) into a 0–5 network‐quality level
func convertGSMSignalToLevel(raw int32) int32 {
	const (
		unknown = 99
		maxRaw  = 31
	)
	if raw == unknown || raw <= 0 {
		return 0
	}
	if raw > maxRaw {
		raw = maxRaw
	}

	level := raw * 6 / (maxRaw + 1)
	if level > 5 {
		level = 5
	}
	return level
}
