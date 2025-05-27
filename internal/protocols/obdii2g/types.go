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
	Raw       string
	IMEI      string
	Timestamp time.Time
	Position  GPSCoordinates
	Vehicle   VehicleData
	OBD       map[string]OBDParameter
	Checksum  byte
	IsValid   bool
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
	flagBinary := fmt.Sprintf("%032b", p.Vehicle.EventFlag)

	// Handle optional ignition field
	if ignition := p.parseBitFlag(flagBinary, 17); ignition {
		vs.Ignition = &ignition
	}
	vs.OverSpeeding = p.parseBitFlag(flagBinary, 2)
	vs.CrashDetection = p.parseBitFlag(flagBinary, 24)
	vs.Towing = p.parseBitFlag(flagBinary, 12)
	vs.UnplugBattery = p.parseBitFlag(flagBinary, 5)
	vs.RashDriving = p.parseBitFlag(flagBinary, 25)
	// Add all 32 flags as per protocol doc table

	return vs
}

func (p *Packet) parseBitFlag(binaryStr string, bitPos int) bool {
	if len(binaryStr) != 32 || bitPos < 0 || bitPos > 31 {
		return false
	}
	return binaryStr[bitPos] == '1'
}

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
	if fuel, exists := p.OBD["012F"]; exists && fuel.Valid {
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
	pkt.Position.Satellites, _ = parseInt32(fields[7])
	pkt.Position.Course, _ = parseFloat(fields[10]) // was [11]
	pkt.Position.Speed, _ = parseFloat(fields[11])  // was [9]
	pkt.Position.HDOP, _ = parseFloat(fields[12])   // was [13]

	// ── Vehicle ──────────────────────────────────────────
	pkt.Vehicle.GSMSignal, _ = parseInt32(fields[8])
	pkt.Vehicle.AccumulatedDist, _ = parseInt32(fields[9]) // was [10]
	pkt.Vehicle.AnalogInput, _ = parseInt32(fields[16])
	pkt.Vehicle.EventFlag, _ = parseUint32(fields[17])
	pkt.Vehicle.ExternalBattery, _ = parseInt32(fields[18])
	pkt.Vehicle.InternalBattery, _ = parseInt32(fields[19])
	pkt.Vehicle.TripTime, _ = parseInt32(fields[20])

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

// parseValue handles OBD PID value decoding
func (o *OBDParameter) parseValue(pidCode string) {
	defer func() {
		if r := recover(); r != nil {
			o.Valid = false
		}
	}()

	hexStr := strings.TrimPrefix(o.RawHex, "0641")
	hexStr = strings.TrimPrefix(hexStr, "067F")
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		o.Valid = false
		return
	}

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
		o.Parsed = float64(data[0])
		o.Unit = "km/h"

	case pidEngineLoad:
		o.Parsed = float64(data[0]) * 100 / 255
		o.Unit = "%"

	case pidCoolantTemp:
		o.Parsed = float64(data[0]) - 40
		o.Unit = "°C"

	case pidIntakeAirTemp:
		o.Parsed = float64(data[0]) - 40
		o.Unit = "°C"

	case pidRunTime:
		if len(data) < 2 {
			o.Valid = false
			return
		}
		o.Parsed = float64(uint16(data[0])<<8 | uint16(data[1]))
		o.Unit = "s"

	case pidDistanceMILOn, pidDistanceSinceClear:
		if len(data) < 2 {
			o.Valid = false
			return
		}
		o.Parsed = float64(uint16(data[0])<<8 | uint16(data[1]))
		o.Unit = "km"

	case pidFuelLevelInput:
		o.Parsed = float64(data[0]) * 100 / 255
		o.Unit = "%"

	case pidMafAirFlow:
		if len(data) < 2 {
			o.Valid = false
			return
		}
		o.Parsed = float64(uint16(data[0])<<8|uint16(data[1])) / 100
		o.Unit = "g/s"

	case pidFuelRailPressure:
		if len(data) < 2 {
			o.Valid = false
			return
		}
		o.Parsed = float64(uint16(data[0])<<8|uint16(data[1])) * 0.079
		o.Unit = "kPa"

	case pidWarmupsSinceClear:
		o.Parsed = float64(data[0])
		o.Unit = "cycles"

	case pidBarometricPressure:
		o.Parsed = float64(data[0])
		o.Unit = "kPa"

	case pidControlModuleVoltage:
		if len(data) < 2 {
			o.Valid = false
			return
		}
		o.Parsed = float64(uint16(data[0])<<8|uint16(data[1])) / 1000
		o.Unit = "V"

	case pidAbsoluteLoadValue:
		if len(data) < 2 {
			o.Valid = false
			return
		}
		raw := uint16(data[0])<<8 | uint16(data[1])
		o.Parsed = float64(raw) * 100 / 255
		o.Unit = "%"

	case pidEngineOilTemp:
		o.Parsed = float64(data[0]) - 40
		o.Unit = "°C"
	// Add all supported PIDs from protocol doc
	default:
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
