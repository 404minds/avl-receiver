// Package queclink receives Queclink @Track ASCII reports: every position-carrying report,
// alarms, heartbeat replies, +BUFF replays and AT commands with their +ACK. The GV200 (vehicle
// tracker), GT500 and GL300/GL300VC/GL300W (personal trackers) have full field layouts; every
// other @Track model is parsed by field scan. HEX mode, SMS and UDP are out of scope.
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

// specByPrefix picks the parsing profile from the model prefix of the protocol-version field.
// What the device announces on the wire wins over what the operator registered: a mis-registered
// device would otherwise be parsed with the wrong field layout (both live gateway units announce
// GL300VC while being registered as GV200). Prefixes come from the model PDFs, cross-checked
// against Traccar's Gl200TextProtocolDecoder PROTOCOL_MODELS (Apache-2.0, © Anton Tananaev).
var specByPrefix = map[string]*modelSpec{
	"04": &gv200Spec, "35": &gv200Spec, // GV200 (prefix drifts with firmware)
	"07": &gt500Spec, // GT500
	"1A": &gl300Spec, "30": &gl300Spec, // GL300
	"28": &gl300Spec, // GL300VC
	"2C": &gl300Spec, // GL300W
}

// nameByPrefix names the Queclink models we hold no field layouts for — logging only; their
// reports are parsed by scanPoint. From Traccar's PROTOCOL_MODELS (Apache-2.0, © Anton Tananaev).
var nameByPrefix = map[string]string{
	"02": "GL200", "06": "GV300", "08": "GMT100", "09": "GV50P", "0F": "GV55",
	"10": "GV55LITE", "11": "GL500", "1F": "GV500", "21": "GL200", "25": "GV300",
	"27": "GV300W", "2D": "GV500VC", "2F": "GV55", "3F": "GMT100", "40": "GL500",
	"41": "GV75W", "42": "GT501", "44": "GL530", "45": "GB100", "4F": "GV56",
	"50": "GV55W", "52": "GL50", "55": "GL50B", "5E": "GV500MAP", "6E": "GV310LAU",
	"BD": "CV200", "C2": "GV600M", "C3": "GL320M", "DC": "GV600MG", "DE": "GL500M",
	"DF": "CV100LG", "F1": "GV350M", "F8": "GV800W", "FC": "GV600W",
	"802004": "GV58LAU", "802005": "GV355CEU", "80201E": "GV30CEU",
}

// prefixOf extracts the model prefix from a protocol-version field: newer models carry a 6-char
// model code (10-char version), everything else the first two characters.
func prefixOf(version string) string {
	if len(version) >= 6 {
		if _, ok := nameByPrefix[version[:6]]; ok {
			return version[:6]
		}
	}
	if len(version) >= 2 {
		return version[:2]
	}
	return ""
}

// spec is the parsing profile of this connection: the announced model when we hold its field
// layouts, else the DeviceType the operator registered, else nil (not a Queclink DeviceType).
func (p *Protocol) spec() *modelSpec {
	if s, ok := specByPrefix[p.versionPrefix]; ok {
		return s
	}
	switch p.DeviceType {
	case types.DeviceType_QUECLINK_GV200:
		return &gv200Spec
	case types.DeviceType_QUECLINK_GT500:
		return &gt500Spec
	case types.DeviceType_QUECLINK_GL300:
		return &gl300Spec
	}
	return nil
}

// layoutTrusted reports whether this connection's field positions are the ones our layout tables
// describe: the device announced a model we hold layouts for, or announced nothing and we fall
// back to the registered type. A known-but-unlayouted model (GT501, GV500, …) is not trusted —
// its Number field is not necessarily where our tables expect it.
func (p *Protocol) layoutTrusted() bool {
	if p.versionPrefix == "" {
		return true
	}
	_, ok := specByPrefix[p.versionPrefix]
	return ok
}

// modelName names a prefix for logs.
func modelName(prefix string) string {
	if s, ok := specByPrefix[prefix]; ok {
		return s.device.String()
	}
	if n, ok := nameByPrefix[prefix]; ok {
		return n + " (no field layouts — positions are located by field scan)"
	}
	return "unknown Queclink model"
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
		if s := p.spec(); s == nil || s.device != types.DeviceType_QUECLINK_GV200 {
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
	p.versionPrefix = prefixOf(fields[1])
	if _, ok := specByPrefix[p.versionPrefix]; ok {
		logger.Sugar().Infof("queclink: imei %s, protocol version %s (prefix %s => %s)", p.Imei, fields[1], p.versionPrefix, modelName(p.versionPrefix))
	} else {
		logger.Sugar().Warnf("queclink: imei %s, protocol version %s: prefix %q => %s; we hold no field layouts for it, so positions are recovered by field scan and tail fields (odometer, battery) are not read", p.Imei, fields[1], p.versionPrefix, modelName(p.versionPrefix))
	}
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
		if want, ok := specByPrefix[p.versionPrefix]; !ok {
			logger.Sugar().Warnf("queclink %s: no field layouts for prefix %q (%s), positions recovered by field scan", p.Imei, p.versionPrefix, modelName(p.versionPrefix))
		} else if want.device != p.DeviceType {
			logger.Sugar().Warnf("queclink %s: version prefix %q says %s but the device is registered as %s; parsing follows the wire", p.Imei, p.versionPrefix, want.device, p.DeviceType)
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
