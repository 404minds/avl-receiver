// queclink-verify decodes raw Queclink @Track lines with the *production* parser
// (internal/protocols/queclink) and prints each raw line next to what the platform
// would store, so a human can confirm every field before a fleet goes live.
//
// Usage:
//
//	go run ./docs/scripts/queclink-verify [-type gv200|gt500|gl300|auto] [file...]   2>/dev/null
//	cat lines.txt | go run ./docs/scripts/queclink-verify 2>/dev/null
//
// Capture lines from a live device on the server (ASCII protocol, so tcpdump is enough):
//
//	ssh bobcat "timeout 300 tcpdump -i any -A -s0 -l 'tcp port 21000'" 2>/dev/null \
//	  | grep -o '+[A-Z]*:GT[^$]*\$' | go run ./docs/scripts/queclink-verify 2>/dev/null
//
// stderr carries the receiver's own zap logs (warnings about unknown headers, field
// count mismatches, field-scan fallbacks) — drop 2>/dev/null to see them.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/404minds/avl-receiver/internal/protocols/queclink"
	"github.com/404minds/avl-receiver/internal/types"
)

// memStore is the receiver's store.Store interface, in memory.
type memStore struct {
	recs  chan *types.DeviceStatus
	resps chan *types.DeviceResponse
}

func (s *memStore) Process(context.Context)                     {}
func (s *memStore) Response(context.Context)                    {}
func (s *memStore) GetProcessChan() chan *types.DeviceStatus    { return s.recs }
func (s *memStore) GetResponseChan() chan *types.DeviceResponse { return s.resps }
func (s *memStore) GetCloseChan() chan bool                     { return nil }
func (s *memStore) GetCloseResponseChan() chan bool             { return nil }

func main() {
	kind := flag.String("type", "auto", "registered device type: gv200, gt500, gl300, or auto (from the protocol version prefix)")
	flag.Parse()

	lines, err := readLines(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "queclink-verify:", err)
		os.Exit(1)
	}
	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "queclink-verify: no @Track lines on stdin or in the given files")
		os.Exit(1)
	}

	st := &memStore{recs: make(chan *types.DeviceStatus, 4096), resps: make(chan *types.DeviceResponse, 256)}
	// One Protocol for the whole run: it carries the per-connection state a real
	// socket would (ignition/motion, GSM level, GT500 battery), so replaying a
	// capture here behaves exactly like the live connection did.
	p := &queclink.Protocol{DeviceType: deviceType(*kind, lines[0])}
	if _, _, err := p.Login(bufio.NewReader(strings.NewReader(lines[0]))); err != nil {
		fmt.Fprintf(os.Stderr, "queclink-verify: first line is not a Queclink message: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("IMEI %s   parsed as %s   (%d lines)\n", p.GetDeviceID(), p.GetDeviceType(), len(lines))
	var records, skipped int
	for i, line := range lines {
		var reply strings.Builder
		_ = p.ConsumeStream(bufio.NewReader(strings.NewReader(line)), &reply, st) // returns io.EOF at the end of the line
		fmt.Printf("\n[%d] %s\n", i+1, line)
		if reply.Len() > 0 {
			fmt.Printf("    server replied: %s\n", reply.String())
		}
		got := drain(st.recs)
		for _, r := range got {
			fmt.Println("    " + describe(r))
		}
		for _, r := range drain(st.resps) {
			fmt.Printf("    command response stored: %s\n", r.Response)
		}
		if len(got) == 0 && reply.Len() == 0 {
			fmt.Println("    -> no position stored (state-only, unsupported report, or no GPS fix — see stderr)")
			skipped++
		}
		records += len(got)
	}
	fmt.Printf("\nSummary: %d messages, %d stored records, %d messages stored nothing\n", len(lines), records, skipped)
}

// describe prints only what the platform actually keeps, in wire order, so it can be
// read straight against the raw line above it.
func describe(r *types.DeviceStatus) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-12s %s  lat %.6f  lon %.6f  speed %.1f km/h  course %.0f  alt %.1f",
		r.MessageType, r.Timestamp.AsTime().Format("2006-01-02 15:04:05"),
		r.Position.Latitude, r.Position.Longitude, r.Position.GetSpeed(), r.Position.Course, r.Position.Altitude)
	if r.Position.Satellites > 0 {
		fmt.Fprintf(&b, "  sats %d", r.Position.Satellites)
	}
	fmt.Fprintf(&b, "\n                 ignition %v", r.VehicleStatus.GetIgnition())
	if r.BatteryLevel > 0 {
		fmt.Fprintf(&b, "  battery %d%%", r.BatteryLevel)
	}
	if r.Odometer > 0 {
		fmt.Fprintf(&b, "  odometer %d km", r.Odometer)
	}
	if r.Temperature != 0 {
		fmt.Fprintf(&b, "  temperature %.3f C", r.Temperature)
	}
	if r.ControlModuleVoltage != 0 {
		fmt.Fprintf(&b, "  supply %.3f V", r.ControlModuleVoltage)
	}
	if r.GsmNetwork >= 0 {
		fmt.Fprintf(&b, "  gsm %d/5", r.GsmNetwork)
	}
	if r.IdentificationId != "" {
		fmt.Fprintf(&b, "  driver %s", r.IdentificationId)
	}
	if al := alarms(r.VehicleStatus); len(al) > 0 {
		fmt.Fprintf(&b, "\n                 ALARMS: %s", strings.Join(al, ", "))
	}
	return b.String()
}

func alarms(v *types.VehicleStatus) []string {
	var out []string
	for _, f := range []struct {
		name string
		on   bool
	}{
		{"sos_button_pressed", v.SosButtonPressed}, {"over_speeding", v.OverSpeeding}, {"towing", v.Towing},
		{"rash_driving", v.RashDriving}, {"harsh_braking", v.HarshBraking}, {"harsh_acceleration", v.HarshAcceleration},
		{"unplug_battery", v.UnplugBattery}, {"excessive_idling", v.ExcessiveIdling}, {"excessive_parking", v.ExcessiveParking},
		{"inputs_triggering", v.InputsTriggering}, {"outputs_triggering", v.OutputsTriggering}, {"fuel_theft", v.FuelTheft},
		{"entering_geofence", v.EntringGeofence}, {"exiting_geofence", v.ExitingGeofence},
		{"battery_low", v.BatteryLow}, {"charging_started", v.ChargingStarted}, {"charging_stopped", v.ChargingStopped},
		{"motion_alert", v.MotionAlert}, {"monitoring_on", v.MonitoringOn}, {"monitoring_off", v.MonitoringOff},
		{"tilt_alert", v.TiltAlert}, {"fall_detected", v.FallDetected}, {"no_motion_alert", v.NoMotionAlert},
		{"self_test", v.SelfTest}, {"welfare_alarm", v.WelfareAlarm}, {"check_in_reminder", v.CheckInReminder},
		{"check_out", v.CheckOut}, {"check_in", v.CheckIn}, {"leave_home", v.LeaveHome}, {"arrive_home", v.ArriveHome},
	} {
		if f.on {
			out = append(out, f.name)
		}
	}
	return out
}

// deviceType honours -type, else reads the model from the protocol version prefix (04/35 =
// GV200, 07 = GT500, 1A/30/28/2C = GL300 family). This only simulates the type registered on
// the platform: since the announced prefix wins, parsing follows the wire either way.
func deviceType(kind, first string) types.DeviceType {
	switch strings.ToLower(kind) {
	case "gv200":
		return types.DeviceType_QUECLINK_GV200
	case "gt500":
		return types.DeviceType_QUECLINK_GT500
	case "gl300":
		return types.DeviceType_QUECLINK_GL300
	}
	if f := strings.Split(first, ","); len(f) > 1 && len(f[1]) >= 2 {
		switch f[1][:2] {
		case "07":
			return types.DeviceType_QUECLINK_GT500
		case "1A", "30", "28", "2C":
			return types.DeviceType_QUECLINK_GL300
		}
	}
	return types.DeviceType_QUECLINK_GV200
}

// readLines pulls every '$'-terminated @Track message out of the input, so a raw
// tcpdump dump, a log file or a hand-written fixture file all work.
func readLines(paths []string) ([]string, error) {
	var raw []byte
	if len(paths) == 0 {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		raw = append(raw, b...)
	}
	var out []string
	for _, chunk := range strings.SplitAfter(string(raw), "$") {
		if i := strings.Index(chunk, "+RESP:"); i >= 0 {
			out = append(out, strings.TrimSpace(chunk[i:]))
		} else if i := strings.Index(chunk, "+BUFF:"); i >= 0 {
			out = append(out, strings.TrimSpace(chunk[i:]))
		} else if i := strings.Index(chunk, "+ACK:"); i >= 0 {
			out = append(out, strings.TrimSpace(chunk[i:]))
		}
	}
	return out, nil
}

func drain[T any](ch chan T) []T {
	var out []T
	for {
		select {
		case v := <-ch:
			out = append(out, v)
		default:
			return out
		}
	}
}
