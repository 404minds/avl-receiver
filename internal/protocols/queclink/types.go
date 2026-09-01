package queclink

// Field layouts come from "GV200 @Track Air Interface Protocol V5.01", "GT500 @Tracker Air
// Interface Protocol V0.13" and "GL300 @Tracker Air Interface Protocol V6.00" (TRACGL300AN008)
// and were checked against real device traffic in Traccar's Gl200TextProtocolDecoderTest corpus
// (Apache-2.0, © Anton Tananaev). The append-mask handling in parsePoint follows Traccar's
// Gl200TextProtocolDecoder.decodeLocation.

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ignitionSpeedKmh: fallback only. Until the device has told us its state, a fix faster than
// this counts as "ignition on" (GPS reports 1–2 km/h of jitter when parked; both devices' own
// movement thresholds default to 5 km/h). The GV200 switches to its real ignition signal at the
// first GTIGN/GTIGF/GTIGL/GTSTT/GTINF; the GT500 has no ignition line, so its motion sensor
// (GTSTT 41/42, GTNMR) plays that role — which needs AT+GTNMD enabled, otherwise the device
// turns motion detection off (state 99) and only the speed rule remains.
const ignitionSpeedKmh = 3.0

// layout says where the GPS blocks sit in a report and what follows them.
type layout struct {
	numIdx int // index of the Number field; -1 = exactly one GPS block
	start  int // index of the first GPS field
	tail   int // fields after the GPS blocks; -1 = variable
	// GV200: the first field after the blocks is Mileage (km).
	// GT500: the tail ends with Odo mileage (km), Battery %, SendTime, Count.
	mileage bool
}

// gt500Layouts: every GT500 report that carries a GPS block (GT500 PDF §3.3.1 and §3.3.5; blocks
// are 11 wide). tail 6 = MAC, Wi-Fi RSSI, Odo, Batt%, SendTime, Count; tail 5 = Reserved, MAC,
// RSSI, SendTime, Count — those reports carry no battery, so the last one seen on the connection
// is reused; tail -1 = GTFLC's variable cell/Wi-Fi scan, which still ends Odo, Batt%, SendTime, Count.
var gt500Layouts = map[string]layout{
	// general position report: …,ReportID,ReportType,Number,N×point,MAC,RSSI,Odo,Batt%,SendTime,Count
	"GTFRI": {6, 7, 6, true}, "GTGEO": {6, 7, 6, true}, "GTSPD": {6, 7, 6, true}, "GTSOS": {6, 7, 6, true},
	"GTRTL": {6, 7, 6, true}, "GTPNL": {6, 7, 6, true}, "GTPOR": {6, 7, 6, true}, "GTNMR": {6, 7, 6, true},
	"GTMSA": {6, 7, 6, true},
	"GTLBC": {-1, 5, 6, true},  // location by call: …,CallNumber,point,…
	"GTGCR": {-1, 7, 6, true},  // geo-fence centre: …,GeoMode,Radius,CheckInterval,point,…
	"GTFLC": {-1, 6, -1, true}, // FRI mode 5: …,Res,Res,point,N×(MCC,MNC,LAC,CID,RSSI,Res),Wi-Fi APs,Odo,Batt%,…
	// …,State|GeoActive|FPOStatus|BatteryV|Reserved,point,Reserved,MAC,RSSI,SendTime,Count
	"GTSTT": {-1, 5, 5, false}, "GTSWG": {-1, 5, 5, false}, "GTFPO": {-1, 5, 5, false},
	"GTBPL": {-1, 5, 5, false}, "GTSTC": {-1, 5, 5, false},
	"GTBTC": {-1, 4, 5, false}, // …,point,Reserved,MAC,RSSI,SendTime,Count
	"GTMON": {-1, 8, 5, false}, // voice monitoring: …,Phone,MONType,Mic,Speaker,point,…
}

// gv200Layouts: every GV200 report that carries a GPS block (GV200 PDF §3.3; blocks are 12 wide
// plus append-mask extras). GTGIN/GTGOT (polygon geo-fence) are deliberately absent: the PDF's
// table and its example disagree on where the block sits, so they go through scanPoint.
var gv200Layouts = map[string]layout{
	"GTFRI": {6, 7, 11, true}, // …,AnalogVCC,ReportIDType,Number,N×point,Mileage,HourMeter,MAV1-3,DI,DO,Res,Res,SendTime,Count
	"GTERI": {7, 8, -1, true}, // GTFRI with the ERI mask inserted at #4 and a variable 1-wire/fuel tail
	"GTTMP": {7, 8, 15, true}, // temperature alarm: GTFRI shape + Res,Res,Res,SensorID,Res,SensorData(°C) before SendTime
	// event reports: …,Reserved|AnalogVCC,ReportIDType(2 hex),Number,point,Mileage,SendTime,Count
	"GTTOW": {6, 7, 3, true}, "GTDIS": {6, 7, 3, true}, "GTIOB": {6, 7, 3, true}, "GTGEO": {6, 7, 3, true},
	"GTSPD": {6, 7, 3, true}, "GTSOS": {6, 7, 3, true}, "GTRTL": {6, 7, 3, true}, "GTDOG": {6, 7, 3, true},
	"GTIGL": {6, 7, 3, true}, "GTHBM": {6, 7, 3, true}, "GTAIS": {6, 7, 3, true}, "GTMAI": {6, 7, 3, true},
	// ignition on/off: …,DurationSec,point,Mileage,HourMeter,SendTime,Count
	"GTIGN": {-1, 5, 4, true}, "GTIGF": {-1, 5, 4, true},
	// idling, start/stop/long stop: …,Reserved|MotionState,Reserved|DurationSec,point,Mileage,SendTime,Count
	"GTIDN": {-1, 6, 3, true}, "GTIDF": {-1, 6, 3, true},
	"GTSTR": {-1, 6, 3, true}, "GTSTP": {-1, 6, 3, true}, "GTLSP": {-1, 6, 3, true},
	// main power on/off, backup battery connected, jamming: …,point,SendTime,Count
	"GTMPN": {-1, 4, 2, false}, "GTMPF": {-1, 4, 2, false}, "GTBTC": {-1, 4, 2, false}, "GTJDR": {-1, 4, 2, false},
	// …,BatteryVCC|Reserved|JammingState|AntennaState|MotionState|CallNumber,point,SendTime,Count
	"GTBPL": {-1, 5, 2, false}, "GTSTC": {-1, 5, 2, false}, "GTJDS": {-1, 5, 2, false}, "GTANT": {-1, 5, 2, false},
	"GTSTT": {-1, 5, 2, false}, "GTLBC": {-1, 5, 2, false},
	// output changed, fuel lost: …,OutputID,Active,point,… / …,InputID,IgnOffFuel%,IgnOnFuel%,point,…
	"GTDOS": {-1, 6, 2, false}, "GTFLA": {-1, 7, 2, false},
	// driver identification (iButton): …,Reserved,ID,IDReportType,Number,point,Mileage,Res×4,SendTime,Count
	"GTIDA": {7, 8, 7, true},
}

// gl300Layouts: every GL300/GL300VC/GL300W report that carries a GPS block (GL300 PDF V6.00
// §3.3). Blocks are 11 GPS fields plus one per-block <Odo mileage> (blockExtra). The general
// position reports end …,Battery%[,I/O status],SendTime,Count — with the last block's odo right
// before that, so the end-anchored [-4] odo / [-3] battery read works for any point count. The
// event reports carry the last known position and end …,SendTime,Count with no battery, so
// mileage stays false there and the cached battery is reused.
var gl300Layouts = map[string]layout{
	// general position report: …,ReportID/AppendMask,ReportType,Number,N×(point,Odo),Batt%,SendTime,Count
	"GTFRI": {6, 7, 3, true}, "GTGEO": {6, 7, 3, true}, "GTSPD": {6, 7, 3, true}, "GTSOS": {6, 7, 3, true},
	"GTRTL": {6, 7, 3, true}, "GTPNL": {6, 7, 3, true}, "GTNMR": {6, 7, 3, true}, "GTDIS": {6, 7, 3, true},
	"GTDOG": {6, 7, 3, true}, "GTPFL": {6, 7, 3, true}, "GTIGL": {6, 7, 3, true},
	"GTLBC": {-1, 5, 3, true}, // location by call: …,CallNumber,point,…
	// events with the last known point: …,(point,Odo),SendTime,Count
	"GTEPN": {-1, 4, 2, false}, "GTEPF": {-1, 4, 2, false}, "GTBTC": {-1, 4, 2, false},
	// GTSTC's Reserved field is zero-length in the PDF's example but a real empty field on some
	// firmware — the field-scan rescue catches the long shape.
	"GTSTC": {-1, 4, 2, false},
	// …,State|BatteryV|GeoActivity,(point,Odo),SendTime,Count
	"GTSTT": {-1, 5, 2, false}, "GTBPL": {-1, 5, 2, false}, "GTSWG": {-1, 5, 2, false},
	// tamper/light switch: …,ReportID,SwitchState,(point,Odo),SendTime,Count (traccar corpus)
	"GTTSW": {-1, 6, 2, false}, "GTLSW": {-1, 6, 2, false},
	// jamming, ignition (GL300VC/GL300W wire), temperature alarm — variable tails
	"GTJDR": {-1, 4, -1, false}, "GTJDS": {-1, 5, -1, false},
	"GTIGN": {-1, 5, -1, false}, "GTIGF": {-1, 5, -1, false},
	"GTTEM": {-1, 6, -1, false}, // …,AlarmID,Temperature(°C),point,…
}

// modelSpec ties one Queclink model family's field layouts together. wide marks the GV200
// conventions (12-wide GPS blocks with an append mask, tail starting with Mileage); the personal
// trackers (GT500, GL300 family) use 11-wide blocks. blockExtra counts fixed fields after each
// GPS block: the GL300 family repeats <Odo mileage> per block (visible in the PDF's two-point
// GTFRI example and in Traccar's GL300W corpus lines), the others put mileage in the tail.
type modelSpec struct {
	device     types.DeviceType
	wide       bool
	blockExtra int
	layouts    map[string]layout
}

var (
	gv200Spec = modelSpec{types.DeviceType_QUECLINK_GV200, true, 0, gv200Layouts}
	gt500Spec = modelSpec{types.DeviceType_QUECLINK_GT500, false, 0, gt500Layouts}
	gl300Spec = modelSpec{types.DeviceType_QUECLINK_GL300, false, 1, gl300Layouts}
)

type point struct {
	speed, course, altitude, lon, lat float32
	satellites                        int32
	time                              time.Time
}

var errNoFix = errors.New("no fix")

func splitFields(line string) []string {
	fields := strings.Split(line, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return fields
}

func atoi(s string) int     { n, _ := strconv.Atoi(s); return n }
func atof(s string) float32 { f, _ := strconv.ParseFloat(s, 32); return float32(f) }

// km converts a mileage string to whole kilometres; anything unparseable or out of int32 range is 0.
func km(s string) int32 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 || f > math.MaxInt32 {
		return 0
	}
	return int32(math.Floor(f))
}

// parsePoint reads one GPS block at fields[i]: HDOP, speed, azimuth, altitude, lon, lat, UTC,
// MCC, MNC, LAC, CellID (11 fields, GT500/GL300) plus for the GV200 an append mask whose bit0
// adds a satellites field and bit1 a trigger-type field. extra counts the model's fixed fields
// after each block (GL300: per-block odo). The width is returned even when the point is unusable
// so the caller can keep walking the message.
func parsePoint(fields []string, i int, gv200 bool, extra int) (point, int, error) {
	width := 11 + extra
	if gv200 {
		width = 12
	}
	if i+width > len(fields) {
		return point{}, width, errors.New("truncated GPS block")
	}
	var pt point
	if gv200 {
		mask := atoi(fields[i+11])
		if mask&1 != 0 {
			if i+width < len(fields) {
				pt.satellites = int32(atoi(fields[i+width]))
			}
			width++
		}
		if mask&2 != 0 {
			width++
		}
	}
	if fields[i+4] == "" || fields[i+5] == "" || fields[i+6] == "" {
		return pt, width, errNoFix
	}
	lon, err1 := strconv.ParseFloat(fields[i+4], 32)
	lat, err2 := strconv.ParseFloat(fields[i+5], 32)
	if err1 != nil || err2 != nil {
		return pt, width, fmt.Errorf("bad coordinates %q,%q", fields[i+4], fields[i+5])
	}
	if lon == 0 && lat == 0 {
		return pt, width, errNoFix
	}
	t, err := time.Parse("20060102150405", fields[i+6])
	if err != nil {
		return pt, width, err
	}
	pt.lon, pt.lat, pt.time = float32(lon), float32(lat), t
	pt.speed, pt.course, pt.altitude = atof(fields[i+1]), atof(fields[i+2]), atof(fields[i+3])
	if pt.speed < 0 { // GT500 sends -1 when speed is unknown
		pt.speed = 0
	}
	return pt, width, nil
}

// parsePoints walks the Number GPS blocks starting at fields[start] (exactly one block when
// numIdx < 0) and returns the usable points plus the index of the first field after the blocks.
func parsePoints(header string, fields []string, numIdx, start int, gv200 bool, extra int) ([]point, int) {
	n := 1
	if numIdx >= 0 {
		n = atoi(fields[numIdx])
	}
	var pts []point
	i := start
	for k := 0; k < n; k++ {
		pt, width, err := parsePoint(fields, i, gv200, extra)
		if i+width > len(fields) {
			logger.Sugar().Warnf("queclink %s: Number=%d but only %d GPS blocks fit in %d fields", header, n, k, len(fields))
			break
		}
		switch {
		case err == nil:
			pts = append(pts, pt)
		case errors.Is(err, errNoFix):
			logger.Sugar().Infof("queclink %s: point %d has no fix, skipped", header, k)
		default:
			logger.Sugar().Warnf("queclink %s: point %d skipped: %v", header, k, err)
		}
		i += width
	}
	return pts, i
}

// coordRe is how a longitude/latitude field looks on the wire: up to 3 integer digits, 6 decimals.
var coordRe = regexp.MustCompile(`^-?\d{1,3}\.\d{6}$`)

// scanPoint finds the GPS block of a report whose layout is not tabled: the first
// longitude,latitude pair followed by a 14-digit UTC time; HDOP, speed, azimuth and altitude are
// the four fields before it (Traccar's Gl200TextProtocolDecoder.decodeBasic does the same).
// Config dumps that contain a geo-fence centre do not match: the field after the pair is a radius.
func scanPoint(fields []string, gv200 bool) (point, bool) {
	for i := 8; i+2 < len(fields); i++ { // the earliest block starts at #4, so lon is #8 at best
		if !coordRe.MatchString(fields[i]) || !coordRe.MatchString(fields[i+1]) || len(fields[i+2]) != 14 {
			continue
		}
		pt, _, err := parsePoint(fields, i-4, gv200, 0)
		return pt, err == nil
	}
	return point{}, false
}

// parseReport turns one +RESP/+BUFF line into zero or more records, updating the connection's
// ignition/motion state, GSM level and (personal trackers) battery on the way. The parsing
// profile is the announced model when we hold its layouts, else the registered device type.
func (p *Protocol) parseReport(fields []string, raw []byte) []*types.DeviceStatus {
	header := fields[0]
	colon := strings.IndexByte(header, ':')
	if colon < 0 {
		return nil
	}
	report := header[colon+1:]
	spec := p.spec()
	if spec == nil {
		logger.Sugar().Warnf("queclink %s: device type %s is not a Queclink model, ignored", header, p.DeviceType)
		return nil
	}
	gv200 := spec.wide

	if (report == "GTSTT" || report == "GTINF") && len(fields) > 4 { // <State> at #4
		p.applyState(fields[4])
	}
	if report == "GTINF" { // device information: no position, but GSM level and (GT500) battery
		p.readInfo(fields, spec)
		return nil
	}

	l, known := spec.layouts[report]
	var pts []point
	next := -1
	scanned := false // position located by scanPoint: the field positions of the rest are unknown
	if known {
		if len(fields) <= l.start {
			logger.Sugar().Warnf("queclink %s: only %d fields, ignored", header, len(fields))
			return nil
		}
		pts, next = parsePoints(header, fields, l.numIdx, l.start, gv200, spec.blockExtra)
		switch {
		// The layout produced nothing. Unless the device explicitly said "0 points", its fields
		// are not where this layout expects them — a model we mis-guessed, or a gateway that
		// reshapes reports (the tstGW leaves the Number field empty on every GTFRI). Locate the
		// block instead of dropping the report; Traccar's decoder never trusts the Number field.
		case len(pts) == 0 && !(p.layoutTrusted() && saysZeroPoints(fields, l.numIdx)):
			if pt, ok := scanPoint(fields, gv200); ok {
				logger.Sugar().Infof("queclink %s: %s does not match the %s layout; position located by field scan", p.Imei, header, modelName(p.versionPrefix))
				pts, scanned = []point{pt}, true
			}
		case l.tail >= 0 && len(fields) != next+l.tail:
			logger.Sugar().Warnf("queclink %s: %d fields, expected %d", header, len(fields), next+l.tail)
		}
	} else {
		pt, ok := scanPoint(fields, gv200)
		if !ok { // no GPS block: config dumps, cell/Wi-Fi scans, power on/off, …
			p.ignore(header, len(fields))
			return nil
		}
		logger.Sugar().Infof("queclink %s: %s is not in the %s layout table; position taken from a field scan", p.Imei, header, modelName(p.versionPrefix))
		pts, scanned = []point{pt}, true
	}

	battery, odometer := p.battery, int32(0)
	var temperature, extVoltage float32
	var driverID string
	if known && l.mileage {
		switch {
		case gv200:
			if !scanned && next < len(fields) {
				odometer = km(fields[next])
			}
		case !scanned && (l.tail < 0 || len(fields) == next+l.tail): // …,Odo,Batt%,SendTime,Count
			odometer = km(fields[len(fields)-4]) // device ODO (km); 0 unless AT+GTCFG ODO enable
			if n, err := strconv.Atoi(fields[len(fields)-3]); err == nil && n >= 0 && n <= 100 {
				battery = int32(n)
				p.battery = battery
			}
		case scanned: // the tail stays anchored at the end even when the middle is reshaped
			// (tstGW), but the odometer unit of such a shape is unverified — battery % only
			if n, err := strconv.Atoi(fields[len(fields)-3]); err == nil && n >= 0 && n <= 100 {
				battery = int32(n)
				p.battery = battery
			}
		default:
			logger.Sugar().Warnf("queclink %s: tail length mismatch, odometer/battery not read", header)
		}
	}
	if gv200 && !scanned {
		switch report {
		case "GTFRI", "GTAIS":
			extVoltage = atof(fields[4]) / 1000 // <Analog Input VCC>: external power, mV
		case "GTERI":
			extVoltage = atof(fields[5]) / 1000
			temperature = eriTemperature(header, fields, next)
		case "GTTMP":
			extVoltage = atof(fields[5]) / 1000
			temperature = atof(fields[len(fields)-3]) // <Temperature Sensor device DATA>, -55…125 °C
		case "GTIDA": // <ID> at #5, <ID Report Type> at #6: 1 authorised, 0 unauthorised / IDA disabled
			driverID = fields[5]
			logger.Sugar().Infof("queclink %s: driver ID %q read, report type %s", p.Imei, driverID, fields[6])
		case "GTIGN":
			p.setIgnition(true)
		case "GTIGF":
			p.setIgnition(false)
		case "GTIGL": // report type 0 = ignition on, 1 = ignition off (GV200 PDF §3.3.1; Traccar has this inverted)
			p.setIgnition(reportType(fields, true) == 0)
		}
	} else if !gv200 && !scanned {
		switch report {
		case "GTNMR": // motion sensor: 0 = motion → rest, 1 = rest → motion (GT500/GL300 PDFs §3.3.1)
			p.setIgnition(reportType(fields, false) == 1)
		case "GTIGN": // GL300VC/GL300W wired ignition
			p.setIgnition(true)
		case "GTIGF":
			p.setIgnition(false)
		case "GTIGL": // report type 0 = ignition on, 1 = ignition off (GL300 PDF §3.3.1)
			p.setIgnition(reportType(fields, false) == 0)
		case "GTTEM":
			temperature = atof(fields[5]) // <Temperature> °C at #5 (GL300 PDF §3.3.4)
		}
	}

	out := make([]*types.DeviceStatus, 0, len(pts))
	for _, pt := range pts {
		rec := toDeviceStatus(p.Imei, p.DeviceType, header, pt, p.ignition(pt.speed), battery, odometer, raw)
		rec.Temperature = temperature
		rec.ControlModuleVoltage = extVoltage // vehicle supply voltage; the consumer has no column for it yet
		rec.IdentificationId = driverID
		rec.GsmNetwork = p.gsmLevel()
		setAlarms(rec.VehicleStatus, report, reportType(fields, gv200))
		out = append(out, rec)
	}
	return out
}

// readInfo caches what a GTINF carries for later position records: <CSQ RSSI> at #6 on every
// model; on the personal trackers (GT500, GL300 family) also <battery percentage>, the 7th field
// from the end (the PDFs' tables and examples disagree on the reserved fields before it, but the
// tail after it is fixed — …,Batt%,FlashType,Temperature,LockState,Reserved,SendTime,Count on
// the GL300, verified against Traccar corpus lines). The GV200 has no battery field here.
func (p *Protocol) readInfo(fields []string, spec *modelSpec) {
	if len(fields) > 6 {
		if lvl, ok := csqLevel(fields[6]); ok {
			p.gsm, p.gsmSeen = lvl, true
		}
	}
	if spec.wide || len(fields) < 12 {
		return
	}
	if n, err := strconv.Atoi(fields[len(fields)-7]); err == nil && n >= 0 && n <= 100 {
		p.battery = int32(n)
	}
}

// saysZeroPoints reports whether the device explicitly announced zero GPS points. An empty field
// is not zero: it means the device did not say, which is why it is worth going and looking.
func saysZeroPoints(fields []string, numIdx int) bool {
	if numIdx < 0 || numIdx >= len(fields) || fields[numIdx] == "" {
		return false
	}
	n, err := strconv.Atoi(fields[numIdx])
	return err == nil && n == 0
}

// eriTemperature returns the first 1-wire temperature sensor reading in a GTERI tail, or 0.
// GV200 PDF §3.3.1: after Mileage, HourMeter, MAV1-3, DI, DO, UART type come [fuel sensor data
// if ERI mask bit0], then [if bit1: <AC100 device count>, count × (<id 16 hex>, <type>, <data>)],
// SendTime, Count. Type 1 = temperature sensor; data is hex, two's complement, × 0.0625 °C
// (e.g. FFE2 → −1.875 °C). Only the first temperature sensor fits DeviceStatus; the rest stay in
// raw_data. The fuel-sensor field has no documented unit and is skipped, not mapped.
func eriTemperature(header string, fields []string, next int) float32 {
	mask, _ := strconv.ParseUint(fields[4], 16, 32)
	if mask&2 == 0 {
		return 0
	}
	i := next + 8 // → AC100 device count
	if mask&1 != 0 {
		i++ // fuel sensor data sits before the 1-wire block
	}
	if i >= len(fields) {
		return 0
	}
	n := atoi(fields[i])
	if n <= 0 {
		return 0
	}
	if len(fields) != i+1+3*n+2 {
		logger.Sugar().Warnf("queclink %s: %d 1-wire devices announced but %d fields, temperature ignored", header, n, len(fields))
		return 0
	}
	for k := 0; k < n; k++ {
		typ, data := fields[i+2+3*k], fields[i+3+3*k]
		if typ != "1" || data == "" {
			continue
		}
		raw, err := strconv.ParseUint(data, 16, 32) // 4 hex on the wire; the table allows 8
		if err != nil {
			continue
		}
		return float32(int16(uint16(raw))) * 0.0625
	}
	return 0
}

// reportType: GT500 sends Report ID and Report Type as two decimal fields (#4, #5); the GV200
// packs them into one 2-hex-char field at #5 (high nibble ID, low nibble type).
func reportType(fields []string, gv200 bool) int {
	if len(fields) < 6 {
		return 0
	}
	if !gv200 {
		return atoi(fields[5])
	}
	v, _ := strconv.ParseUint(fields[5], 16, 8)
	return int(v & 0xF)
}

// setAlarms maps an event report onto the VehicleStatus flags the consumer turns into events.
// GTSPD is only sent when the configured speed alarm fires (AT+GTSPD mode 1 = inside the
// range, mode 2 = outside), so every GTSPD is an over-speed report whatever its type says.
func setAlarms(vs *types.VehicleStatus, report string, reportType int) {
	switch report {
	case "GTSOS":
		vs.SosButtonPressed = true
	case "GTSPD":
		vs.OverSpeeding = true
	case "GTGEO": // 0 = exit, 1 = enter
		if reportType == 1 {
			vs.EntringGeofence = true
		} else {
			vs.ExitingGeofence = true
		}
	case "GTTOW":
		vs.Towing = true
	case "GTHBM": // 0 = harsh braking, 1 = harsh acceleration
		vs.RashDriving = true
		switch reportType {
		case 0:
			vs.HarshBraking = true
		case 1:
			vs.HarshAcceleration = true
		}
	case "GTMPF": // main power cut
		vs.UnplugBattery = true
	case "GTIDN": // idling started
		vs.ExcessiveIdling = true
	case "GTDIS": // digital input changed
		vs.InputsTriggering = true
	case "GTDOS": // digital output changed
		vs.OutputsTriggering = true
	case "GTFLA": // fuel level after ignition on is lower than before the last ignition off by more than AT+GTFLA's threshold
		vs.FuelTheft = true
	case "GTLSP": // stopped longer than AT+GTSTR <Long Stop>
		vs.ExcessiveParking = true
	case "GTGIN": // entered a polygon geo-fence
		vs.EntringGeofence = true
	case "GTGOT": // left a polygon geo-fence
		vs.ExitingGeofence = true
	}
}

// gsmLevel returns the platform's 0–5 signal scale (UI: 0 No Network … 5 Excellent) from the last
// GTINF, or -1 (UI "N/A") before the device has reported one. GTINF is sent at power-on and on
// the AT+GTCFG info-report interval; position reports carry no GSM level (the GT500's
// "Signal Strength" is the Wi-Fi access point's).
func (p *Protocol) gsmLevel() int32 {
	if !p.gsmSeen {
		return -1
	}
	return p.gsm
}

// csqLevel maps a <CSQ RSSI> (0–31, 99 unknown; dBm = -113 + 2×CSQ per the GV200 PDF table) onto
// the UI's five labels. Thresholds ≈ -65/-75/-85/-95 dBm.
func csqLevel(csq string) (int32, bool) {
	n, err := strconv.Atoi(csq)
	if err != nil || n < 0 || n > 31 {
		return 0, false // empty, 99 or garbage
	}
	switch {
	case n == 0:
		return 0, true
	case n >= 24:
		return 5, true
	case n >= 19:
		return 4, true
	case n >= 14:
		return 3, true
	case n >= 9:
		return 2, true
	default:
		return 1, true
	}
}

// ignition returns the device's own signal once one has been seen on this connection (GV200:
// ignition line; GT500: motion sensor), else the speed rule.
func (p *Protocol) ignition(speed float32) bool {
	if p.lastIgnition != nil {
		return *p.lastIgnition
	}
	return speed > ignitionSpeedKmh
}

func (p *Protocol) setIgnition(on bool) { p.lastIgnition = &on }

// applyState reads the GTSTT/GTINF <State>: 11/12 ignition off (rest/motion), 21/22 ignition on,
// 16/1A towing. 41/42 = motion sensor rest/motion with no ignition signal: on the personal
// trackers (GT500, GL300 family) that is the movement signal we use; on a GV200 it means the
// ignition line is not wired, so the speed rule stays. 99 = the device switched motion detection
// off → forget the state, back to the speed rule.
func (p *Protocol) applyState(state string) {
	personal := false
	if s := p.spec(); s != nil {
		personal = !s.wide
	}
	switch state {
	case "11", "12":
		p.setIgnition(false)
	case "21", "22":
		p.setIgnition(true)
	case "41", "42":
		if personal {
			p.setIgnition(state == "42")
		}
	case "99":
		if personal {
			p.lastIgnition = nil
		}
	}
}

func toDeviceStatus(imei string, dt types.DeviceType, header string, pt point, ignition bool, battery, odometer int32, raw []byte) *types.DeviceStatus {
	speed := pt.speed
	return &types.DeviceStatus{
		Imei:        imei,
		DeviceType:  dt,
		Timestamp:   timestamppb.New(pt.time),
		MessageType: strings.TrimLeft(header, "+"), // RESP:GTFRI, BUFF:GTFRI, ...
		Position: &types.GPSPosition{
			Latitude:   pt.lat,
			Longitude:  pt.lon,
			Altitude:   pt.altitude,
			Speed:      &speed,
			Course:     pt.course,
			Satellites: pt.satellites,
		},
		VehicleStatus: &types.VehicleStatus{Ignition: &ignition},
		BatteryLevel:  battery,
		Odometer:      odometer,
		RawData:       &types.DeviceStatus_QueclinkPacket{QueclinkPacket: &types.QueclinkPacket{RawData: raw}},
	}
}
