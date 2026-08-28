// Package queclink receives Queclink @Track ASCII reports (GV200 vehicle tracker, GT500
// personal tracker): every position-carrying report, alarms, heartbeat replies, +BUFF replays and
// AT commands with their +ACK. HEX mode, SMS and UDP are out of scope.
package queclink

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
)

var logger = configuredLogger.Logger

// modelByPrefix maps the first two characters of the protocol-version field to a model.
// Informational only: dispatch is on the DeviceType the operator registered, because the prefix
// drifts with firmware (GV200 units ship as both 04 and 35).
var modelByPrefix = map[string]types.DeviceType{
	"04": types.DeviceType_QUECLINK_GV200,
	"35": types.DeviceType_QUECLINK_GV200,
	"07": types.DeviceType_QUECLINK_GT500,
}

// sendGenericSack: reply "+SACK:<count>$" to every report. Only flip when devices are provisioned
// with SACK Enable = 1 (AT+GTSRI / AT+GTQSS); with SACK Enable = 0 the device ignores it.
const sendGenericSack = false

// binaryHeaders are the @Track "HEX mode" frame types; the same device sends these instead of
// "+RESP:" when Protocol Format is 1. We do not decode HEX mode.
var binaryHeaders = []string{"+RSP", "+EVT", "+INF", "+HBD", "+BSP", "+BVT", "+BNF", "+ACK", "+LGN", "+CRD"}

type Protocol struct {
	DeviceType    types.DeviceType
	Imei          string
	versionPrefix string
	lastIgnition  *bool          // device state seen on this connection (GV200 ignition line, GT500 motion sensor); nil = speed rule
	ignored       map[string]int // headers seen but not decoded on this connection, with counts
	serial        int            // commands sent on this connection (log/debug only)
	gsm           int32          // GSM level 0–5 from the last GTINF <CSQ RSSI>; valid when gsmSeen
	gsmSeen       bool
	battery       int32 // GT500 battery % from the last report that carried one (GTFRI-shaped tail or GTINF); reused by reports without it
}

// serialCounter numbers outgoing commands (4 hex digits, wraps); the device echoes it in +ACK.
var serialCounter atomic.Uint32

func commandSerial() uint32 { return serialCounter.Add(1) & 0xFFFF }

func (p *Protocol) GetDeviceID() string              { return p.Imei }
func (p *Protocol) GetDeviceType() types.DeviceType  { return p.DeviceType }
func (p *Protocol) SetDeviceType(t types.DeviceType) { p.DeviceType = t }
func (p *Protocol) GetProtocolType() types.DeviceProtocolType {
	return types.DeviceProtocolType_QUECLINK
}

// gv200Password is the device password every AT command starts with (factory default "gv200",
// GV200 PDF §3.2). Fleets that changed it must send custom commands with their own password.
const gv200Password = "gv200"

// immobiliserOutput is the GV200 digital output the immobiliser relay is wired to, and
// immobiliserActive its level while the engine is cut. Installer-specific — confirm on the bench
// (runbook §8) before trusting the ignition_on/ignition_off presets.
const immobiliserOutput, immobiliserActive = 1, 1

// SendCommandToDevice writes one @Track command. The platform's preset keys "ignition_off" /
// "ignition_on" become an AT+GTOUT on the immobiliser output (GV200 only; the GT500 has no
// outputs); anything else is passed through verbatim with the '$' terminator added if missing.
// The device answers "+ACK:GTxxx,…$", which handleLine forwards as a DeviceResponse.
func (p *Protocol) SendCommandToDevice(writer io.Writer, command string) error {
	cmd := strings.TrimSpace(command)
	switch cmd {
	case "ignition_off", "ignition_on":
		if p.DeviceType != types.DeviceType_QUECLINK_GV200 {
			return fmt.Errorf("queclink: %s has no controllable output for %q", p.DeviceType, cmd)
		}
		level := immobiliserActive
		if cmd == "ignition_on" {
			level = 1 - immobiliserActive
		}
		cmd = gtout(level)
	case "":
		return fmt.Errorf("queclink: empty command")
	}
	if !strings.HasSuffix(cmd, "$") {
		cmd += "$"
	}
	p.serial++
	logger.Sugar().Infof("queclink %s: sending %q", p.Imei, cmd)
	_, err := io.WriteString(writer, cmd)
	return err
}

// gtout builds AT+GTOUT setting only the immobiliser output (wave shape 1, others untouched:
// status 0 / duration 0 / toggle 0), no long operation, no GTDOS report.
// Layout (GV200 PDF §3.2): pwd, 4×(status,duration,toggle), long op, DOS report, res, res, serial.
func gtout(level int) string {
	outs := []string{"0,0,0", "0,0,0", "0,0,0", "0,0,0"}
	outs[immobiliserOutput-1] = fmt.Sprintf("%d,0,0", level)
	return fmt.Sprintf("AT+GTOUT=%s,%s,,,,,%04X$", gv200Password, strings.Join(outs, ","), commandSerial())
}

// Login only sniffs: it peeks at the first message, records the IMEI and consumes nothing, so
// ConsumeStream still sees the whole first message.
func (p *Protocol) Login(reader *bufio.Reader) ([]byte, int, error) {
	head, err := reader.Peek(6)
	if err != nil {
		return nil, 0, errs.ErrUnknownProtocol // too short to be ours; let the sniff chain finish
	}
	if !bytes.HasPrefix(head, []byte("+RESP:")) && !bytes.HasPrefix(head, []byte("+BUFF:")) && !bytes.HasPrefix(head, []byte("+ACK:G")) {
		if head[0] == '+' {
			for _, h := range binaryHeaders {
				if bytes.HasPrefix(head, []byte(h)) && head[4] != ':' {
					logger.Sugar().Errorf("queclink: HEX-mode header % X — device is in @Track HEX mode; reprovision Protocol Format 0 (ASCII) via AT+GTSRI/AT+GTQSS", head)
					break
				}
			}
		}
		return nil, 0, errs.ErrUnknownProtocol
	}
	peeked, err := reader.Peek(48) // header, 10-char version and IMEI always fit in 48 bytes
	if err != nil && !errors.Is(err, bufio.ErrBufferFull) && len(peeked) == 0 {
		return nil, 0, fmt.Errorf("queclink: header peek failed: %w", err)
	}
	fields := strings.Split(string(peeked), ",")
	if len(fields) < 4 || !isIMEI(fields[2]) {
		return nil, 0, fmt.Errorf("%w: queclink header %q", errs.ErrBadPacket, peeked)
	}
	p.Imei = fields[2]
	if len(fields[1]) >= 2 {
		p.versionPrefix = fields[1][:2]
	}
	logger.Sugar().Infof("queclink: imei %s, protocol version %s (prefix %s => %s)", p.Imei, fields[1], p.versionPrefix, modelByPrefix[p.versionPrefix])
	return []byte{}, 0, nil
}

func isIMEI(s string) bool {
	if len(s) != 15 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// ConsumeStream frames on '$' (messages carry no length or CRC in ASCII mode), replies to
// heartbeats and forwards every usable GPS point as one record.
func (p *Protocol) ConsumeStream(reader *bufio.Reader, writer io.Writer, st store.Store) error {
	defer func() {
		if len(p.ignored) > 0 {
			logger.Sugar().Infof("queclink %s: headers seen but not decoded on this connection: %v", p.Imei, p.ignored)
		}
	}()
	if p.versionPrefix != "" {
		if want, ok := modelByPrefix[p.versionPrefix]; !ok {
			logger.Sugar().Warnf("queclink %s: unknown version prefix %q, parsing as %s", p.Imei, p.versionPrefix, p.DeviceType)
		} else if want != p.DeviceType {
			logger.Sugar().Warnf("queclink %s: version prefix %q suggests %s but device is registered as %s; using the registered type", p.Imei, p.versionPrefix, want, p.DeviceType)
		}
	}
	for {
		line, err := reader.ReadSlice('$')
		if err != nil {
			if errors.Is(err, bufio.ErrBufferFull) {
				logger.Sugar().Warnf("queclink %s: no '$' within %d bytes, dropping connection", p.Imei, len(line))
				return errs.ErrBadPacket
			}
			return err // io.EOF included; a partial trailing message is dropped
		}
		raw := bytes.Trim(line[:len(line)-1], "\r\n\x00 ")
		if len(raw) == 0 {
			continue
		}
		if err := p.handleLine(append([]byte(nil), raw...), writer, st); err != nil { // copy: ReadSlice reuses its buffer
			return err
		}
	}
}

func (p *Protocol) handleLine(raw []byte, writer io.Writer, st store.Store) error {
	fields := splitFields(string(raw))
	header := fields[0]
	switch {
	case header == "+ACK:GTHBD":
		if len(fields) < 3 {
			logger.Sugar().Warnf("queclink %s: malformed heartbeat %q", p.Imei, raw)
			return nil
		}
		_, err := fmt.Fprintf(writer, "+SACK:GTHBD,%s,%s$", fields[1], fields[len(fields)-1]) // echo version and count verbatim
		return err
	case strings.HasPrefix(header, "+RESP:"), strings.HasPrefix(header, "+BUFF:"):
		for _, rec := range p.parseReport(fields, raw) {
			st.GetProcessChan() <- rec
		}
		if sendGenericSack {
			if _, err := fmt.Fprintf(writer, "+SACK:%s$", fields[len(fields)-1]); err != nil {
				return err
			}
		}
	case strings.HasPrefix(header, "+ACK:GT"): // reply to a command we (or an SMS) sent, e.g. +ACK:GTOUT,…,<serial>,<sendtime>,<count>
		st.GetResponseChan() <- &types.DeviceResponse{Imei: p.Imei, Response: string(raw)}
	default: // anything unexpected
		p.ignore(header, len(fields))
	}
	return nil
}

// ignore counts a header this connection does not decode; logged once per header, summarised on close.
func (p *Protocol) ignore(header string, nfields int) {
	if p.ignored == nil {
		p.ignored = map[string]int{}
	}
	if p.ignored[header] == 0 {
		logger.Sugar().Infof("queclink %s: ignoring %q (%d fields); further ones are counted, not logged", p.Imei, header, nfields)
	}
	p.ignored[header]++
}
