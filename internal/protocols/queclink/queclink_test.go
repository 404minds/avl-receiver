package queclink

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	errs "github.com/404minds/avl-receiver/internal/errors"
	"github.com/404minds/avl-receiver/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixtures. "doc" lines are verbatim from the Queclink protocol PDFs (extraction spaces removed);
// "traccar" lines are real device traffic from Traccar's Gl200TextProtocolDecoderTest corpus
// (Apache-2.0, © Anton Tananaev); "community" is a forum-reported line of unverified origin.
const (
	imei         = "135790246811220"
	gt500, gv200 = types.DeviceType_QUECLINK_GT500, types.DeviceType_QUECLINK_GV200

	// GT500 (doc)
	f1    = "+RESP:GTFRI,072002,135790246811220,,0,0,1,1,4.3,72,90.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,000ce7000000,-56,1000,80,20090214093254,11F0$"
	f2    = "+BUFF:GTFRI,020102,135790246811220,,0,0,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,,20090214093254,11F0$" // 22 fields: doc example contradicts its own table
	f5    = "+ACK:GTHBD,070002,135790246811220,,20100214093254,11F0$"
	f6    = "+RESP:GTPNA,070002,135790246811220,,20100214093254,11F0$"
	gtStt = "+RESP:GTSTT,070002,135790246811220,,41,0,70.0,,,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,000ce7000000,-60,20100214093254,11F0$"
	gtBpl = "+RESP:GTBPL,070002,135790246811220,,3.53,0,4.3,,,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,000ce7000000,-60,20100214093254,11F0$"
	gtInf = "+RESP:GTINF,070002,135790246811220,,41,898600810906F8048812,16,0,,500,,4.10,0,1,0,,,,20100214013254,80,,,,,20100214093254,11F0$"
	f7    = "+RESP:GTFRI,072002,135790246811220,,0,0,2,1,4.3,72,90.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,1,4.3,72,90.0,121.354335,31.222073,20090214013354,0460,0000,18d8,6141,000ce7000000,-56,1000,80,20090214093254,11F0$" // synthetic N=2

	// GV200 (doc)
	gv1      = "+RESP:GTFRI,040100,135790246811220,,,10,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,12345:12:34,,,,00,00,,,20090214093254,11F0$"
	gv2      = "+RESP:GTFRI,040100,135790246811220,,,10,2,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,0,4.3,92,70.0,121.354335,31.222073,20090101000000,0460,0000,18d8,6141,00,2000.0,12345:12:34,,,,00,00,,,20090214093254,11F0$"
	gvEri    = "+RESP:GTERI,040408,862170010903183,,00000001,,10,1,1,0.5,55,71.8,117.200986,31.832871,20120818041519,0460,0000,5663,5A02,00,0.0,,,,,,,,1,0097,20120818041528,0030$"
	gvTow    = "+RESP:GTTOW,040100,135790246811220,,,00,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	gvIgn    = "+RESP:GTIGN,040100,135790246811220,,1200,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,,20090214093254,11F0$"
	gvIgf    = "+RESP:GTIGF,040100,135790246811220,,1200,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,12345:12:34,20090214093254,11F0$"
	gvInf421 = "+RESP:GTINF,040100,135790246811220,,16,898600810906F8048812,16,0,1,,0,4.4,0,0,0,0,20090214013254,13000,00,00,+0800,0,20090214093254,11F0$"

	// GV200 (traccar / community)
	gv517      = "+RESP:GTFRI,04040C,359231038939904,,,10,1,2,0.0,117,346.0,8.924243,50.798077,20130618122040,0262,0002,0299,109C,00,0.0,,,,,,,,,20130618122045,00F6$"
	gvF16      = "+RESP:GTFRI,350302,867844003012625,,12372,10,1,0,0.0,0,820.8,-70.514872,-33.361021,20160811154617,0730,0002,7410,C789,00,0.0,00000:15:30,2788,705,164,0D,00,,,20160811154651,061D$" // queclink-parser README
	gvF17      = "+RESP:GTFRI,04040D,861074022094042,,,10,1,0,,,,0,0,,,,,,,,,,,,00,00,,,,0099$"                                                                                                       // community, no fix
	gvEri2024  = "+RESP:GTERI,358803,862364030335589,gv200,00000002,,10,1,1,42.5,132,2150.4,43.516551,17.661988,20240818171932,0420,0004,F23D,31EF,00,17363.6,,,91,,00,00,2,0,20240818171940,91F1$"
	gvEri2021  = "+RESP:GTERI,040A00,862894022579562,gv200,00000002,,10,1,1,96.1,180,749.7,39.222692,24.165463,20210225065756,0420,0004,759C,3360,00,15529.8,,,2789,,01,00,2,2,282BD47A0B000063,1,FFE2,281FDD5D0B000057,1,FFC8,20210225065800,6974$"
	gvBuffEri  = "+BUFF:GTERI,358803,862364030261132,gv200,00000002,,10,1,1,43.4,160,592.2,46.723488,24.590880,20240817131155,0420,0001,01AE,393D,00,19273.0,,,12,,05,00,2,0,20240817131205,C81A$"
	gvBuffEri2 = "+BUFF:GTERI,358803,862364030181926,gv200,00000011,,51,1,1,21.0,98,30.2,38.249517,24.008269,20240817131401,0420,0004,75A0,412C,00,84795.4,,,,,01,00,0,,20240817131358,F127$"
	gvStt520   = "+RESP:GTSTT,04040C,359231038939904,,42,0,0.0,117,346.0,8.924243,50.798077,20130618125152,0262,0002,0299,109C,00,20130618125154,017A$"
	gvInf307   = "+RESP:GTINF,04040E,861074023747143,gv200,41,8959301000648637556f,24,0,1,0,1,4.4,0,1,0,0,20170912221854,0,00,01,-0500,1,20170912193448,1D5B$"

	// Other Queclink models (traccar) — robustness only: append mask, 10-char version, 8-hex cell id
	f18 = "+RESP:GTFRI,DF0200,868487004353181,cv100,14051,10,1,0,0.0,0,264.1,114.015515,22.537178,20210608064328,0460,0001,25F8,061A7D02,,0.0,,,,100,21,,,,20210608144354,32DB$"
	f19 = "+RESP:GTERI,6E0A03,868589060745174,,00000100,,10,1,1,0.0,0,1509.0,-90.544928,14.584461,20250307182819,0704,0001,13A7,000B60AB,01,12,358.6,,,,,100,1A0000,0,2,00,6,5,0E7109BB,283F,SENTEMP1,7805412CF2DD,1,3466,24,36,24.91,100,01,6,2,,083F,SENTEMP2,7805412CF25C,0,,,,,20250307182820,1ABC$"
	f20 = "+BUFF:GTFRI,8020040200,866314060249032,,12194,10,1,3,0.0,0,20.1,-71.596533,-33.524718,20230926200338,0730,0001,772A,052B253E,02,0,0.0,,,,,0,420000,,,,20230926200340,1549$"
	f21 = "+RESP:GTERI,C30209,860201067200000,,00000080,0,16,1,1,47.2,245,169.3,-122.234955,47.906141,20260131234254,0310,0260,2CA2,014A2E17,,27,,20260131234255,51D8$"
)

var allFixtures = []string{f1, f2, f5, f6, gtStt, gtBpl, gtInf, f7, gv1, gv2, gvEri, gvTow, gvIgn, gvIgf, gvInf421,
	gv517, gvF16, gvF17, gvEri2024, gvEri2021, gvBuffEri, gvBuffEri2, gvStt520, gvInf307, f18, f19, f20, f21}

// ---- scaffolding (§15.1) ----

type fakeStore struct {
	ch   chan *types.DeviceStatus
	resp chan *types.DeviceResponse
}

func newFakeStore() *fakeStore {
	return &fakeStore{ch: make(chan *types.DeviceStatus, 256), resp: make(chan *types.DeviceResponse, 16)}
}
func (s *fakeStore) Process(context.Context)                     {}
func (s *fakeStore) Response(context.Context)                    {}
func (s *fakeStore) GetProcessChan() chan *types.DeviceStatus    { return s.ch }
func (s *fakeStore) GetResponseChan() chan *types.DeviceResponse { return s.resp }
func (s *fakeStore) GetCloseChan() chan bool                     { return nil }
func (s *fakeStore) GetCloseResponseChan() chan bool             { return nil }
func (s *fakeStore) drain() (out []*types.DeviceStatus) {
	for {
		select {
		case r := <-s.ch:
			out = append(out, r)
		default:
			return out
		}
	}
}

// runStream feeds input through ConsumeStream from an in-memory reader; the stream ends with io.EOF.
func runStream(t *testing.T, dt types.DeviceType, input string) (records []*types.DeviceStatus, written string, err error) {
	t.Helper()
	p := &Protocol{DeviceType: dt, Imei: imei}
	st := newFakeStore()
	var w bytes.Buffer
	err = p.ConsumeStream(bufio.NewReader(strings.NewReader(input)), &w, st)
	return st.drain(), w.String(), err
}

func mustStream(t *testing.T, dt types.DeviceType, input string) ([]*types.DeviceStatus, string) {
	t.Helper()
	recs, w, err := runStream(t, dt, input)
	require.ErrorIs(t, err, io.EOF)
	return recs, w
}

// runPipe writes chunks over a real net.Pipe so ConsumeStream sees genuine fragmentation.
func runPipe(t *testing.T, dt types.DeviceType, chunks []string, gap time.Duration) ([]*types.DeviceStatus, string) {
	t.Helper()
	client, server := net.Pipe()
	p := &Protocol{DeviceType: dt, Imei: imei}
	st := newFakeStore()
	var w bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- p.ConsumeStream(bufio.NewReader(server), &w, st) }()
	for _, c := range chunks {
		_, err := client.Write([]byte(c))
		require.NoError(t, err)
		time.Sleep(gap)
	}
	require.NoError(t, client.Close())
	require.ErrorIs(t, <-done, io.EOF)
	return st.drain(), w.String()
}

// parseLine runs the parser only (no framing) on one '$'-terminated line.
func parseLine(t *testing.T, dt types.DeviceType, line string) []*types.DeviceStatus {
	t.Helper()
	raw := strings.TrimSuffix(line, "$")
	return (&Protocol{DeviceType: dt, Imei: imei}).parseReport(splitFields(raw), []byte(raw))
}

func fieldsOf(line string) []string { return strings.Split(strings.TrimSuffix(line, "$"), ",") }
func join(f []string) string        { return strings.Join(f, ",") + "$" }

func setField(line string, i int, v string) string {
	f := fieldsOf(line)
	f[i] = v
	return join(f)
}

func insertAfter(line string, i int, vals ...string) string {
	f := fieldsOf(line)
	out := append(append(append([]string{}, f[:i+1]...), vals...), f[i+1:]...)
	return join(out)
}

func withHeader(line, header string) string { return setField(line, 0, header) }

func utc(y int, mo time.Month, d, h, mi, s int) time.Time {
	return time.Date(y, mo, d, h, mi, s, 0, time.UTC)
}

// ---- §15.3 Login ----

func TestLogin(t *testing.T) {
	const anyErr = "any"
	cases := []struct {
		name, in, wantImei string
		wantErr            interface{} // nil, errs.ErrUnknownProtocol, or anyErr (rejected with a non-sniff error)
	}{
		{"LG1 +RESP", f1, imei, nil},
		{"LG2 +BUFF first", f2, imei, nil},
		{"LG3 heartbeat first", f5, imei, nil},
		{"LG4 +ACK:GTQSS", "+ACK:GTQSS,070002,135790246811220,,0000$", imei, nil},
		{"LG5 aquila", "$$CLIENT_1NS,867322035135914,15,1.1,20160311,155111,1,1,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0*", "", errs.ErrUnknownProtocol},
		{"LG6 gt06", "\x78\x78\x0d\x01\x03\x53\x41\x35\x32\x15\x03\x62\x00\x02\x2d\x06\x0d\x0a", "", errs.ErrUnknownProtocol},
		{"LG6 teltonika", "\x00\x0f353413532150362", "", errs.ErrUnknownProtocol},
		{"LG6 intellitrac", "\xfa\xf8\x00\x02\x00\x00\x00\x00\x00\x00\x00\x00", "", errs.ErrUnknownProtocol},
		{"LG7 fewer than 6 bytes", "+RES", "", errs.ErrUnknownProtocol},
		{"UN4 binary +RSP", "+RSP\x0b\xff\xff\xff\xbf\x00\x5c\x45\x01\x01\x01\x08\x56\x32\x54\x03", "", errs.ErrUnknownProtocol},
		{"LG8 14-digit imei", setField(f1, 2, "13579024681122"), "", anyErr},
		{"LG8 16-digit imei", setField(f1, 2, "1357902468112201"), "", anyErr},
		{"LG8 14-hex id", setField(f1, 2, "A1000043D20139"), "", anyErr},
		{"LG9 10-char version", f20, "866314060249032", nil},
		{"LG10 name gv200", gvEri2024, "862364030335589", nil},
		{"LG10 17-char name", setField(f1, 3, "1G1JC5444R7252367"), imei, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := &Protocol{}
			ack, skip, err := p.Login(bufio.NewReader(strings.NewReader(c.in)))
			switch c.wantErr {
			case nil:
				require.NoError(t, err)
				assert.Equal(t, []byte{}, ack, "ack must be empty but non-nil")
				assert.Equal(t, 0, skip)
				assert.Equal(t, c.wantImei, p.GetDeviceID())
			case anyErr:
				require.Error(t, err)
				assert.NotErrorIs(t, err, errs.ErrUnknownProtocol, "a malformed Queclink header must reject, not decline")
			default:
				assert.ErrorIs(t, err, errs.ErrUnknownProtocol)
			}
		})
	}
}

func TestLoginDoesNotConsume(t *testing.T) { // LG11
	r := bufio.NewReader(strings.NewReader(f1 + f5))
	p := &Protocol{}
	_, _, err := p.Login(r)
	require.NoError(t, err)
	line, err := r.ReadSlice('$')
	require.NoError(t, err)
	assert.Equal(t, f1, string(line))
}

func TestLoginPrefixMismatchFollowsWire(t *testing.T) { // LG12
	r := bufio.NewReader(strings.NewReader(f1)) // prefix 07 = GT500
	p := &Protocol{}
	_, _, err := p.Login(r)
	require.NoError(t, err)
	p.SetDeviceType(gv200) // operator registered it as a GV200
	st := newFakeStore()
	require.ErrorIs(t, p.ConsumeStream(r, io.Discard, st), io.EOF)
	recs := st.drain()
	require.Len(t, recs, 1)
	assert.Equal(t, gv200, recs[0].DeviceType)       // the record keeps the registered type…
	assert.Equal(t, int32(80), recs[0].BatteryLevel) // …but the fields are read as the wire's GT500
	assert.Equal(t, int32(1000), recs[0].Odometer)
}

func TestProtocolIdentity(t *testing.T) {
	p := &Protocol{}
	assert.Equal(t, types.DeviceProtocolType_QUECLINK, p.GetProtocolType())
	p.SetDeviceType(gt500)
	assert.Equal(t, gt500, p.GetDeviceType())
	assert.NoError(t, p.SendCommandToDevice(io.Discard, "AT+GTRTO"))
}

// ---- §15.2 Framing ----

func TestFraming(t *testing.T) {
	f1b := strings.Replace(f1, ",11F0$", ",11F1$", 1)
	cases := []struct {
		name, in string
		wantN    int
		wantSack string
	}{
		{"FR1 single", f1, 1, ""},
		{"FR2 concatenated", f1 + f1b, 2, ""},
		{"FR4 CRLF between", f1 + "\r\n" + f1b, 2, ""},
		{"FR5 NUL padding", f1 + "\x00\x00" + f1b, 2, ""},
		{"FR6 garbage mid-stream", f1 + "xyz$" + f1b, 2, ""},
		{"FR8 missing $ merges A into B", strings.TrimSuffix(f1, "$") + f1b, 1, ""},
		{"FR9 EOF mid-message", f1[:40], 0, ""},
		{"FR12 heartbeat + report", f5 + f1, 1, "+SACK:GTHBD,070002,11F0$"},
		{"empty stream", "", 0, ""},
		{"lone $", "$", 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recs, w := mustStream(t, gt500, c.in)
			assert.Len(t, recs, c.wantN)
			assert.Equal(t, c.wantSack, w)
		})
	}

	t.Run("FR2 order preserved", func(t *testing.T) {
		recs, _ := mustStream(t, gt500, f1+setField(f1, 13, "20090214013354"))
		require.Len(t, recs, 2)
		assert.True(t, recs[0].Timestamp.AsTime().Before(recs[1].Timestamp.AsTime()))
	})
	t.Run("FR10 raw data survives buffer reuse", func(t *testing.T) {
		recs, _ := mustStream(t, gt500, f1+f1b)
		require.Len(t, recs, 2)
		assert.Equal(t, strings.TrimSuffix(f1, "$"), string(recs[0].GetQueclinkPacket().GetRawData()))
		assert.Equal(t, strings.TrimSuffix(f1b, "$"), string(recs[1].GetQueclinkPacket().GetRawData()))
	})
	t.Run("FR11 100 back-to-back", func(t *testing.T) {
		var b strings.Builder
		for i := 0; i < 100; i++ {
			b.WriteString(setField(f1, 23, fmt.Sprintf("%04X", i)))
		}
		recs, _ := mustStream(t, gt500, b.String())
		assert.Len(t, recs, 100)
	})
	t.Run("FR7 no $ within buffer", func(t *testing.T) {
		_, _, err := runStream(t, gt500, strings.Repeat("A", 5000))
		assert.ErrorIs(t, err, errs.ErrBadPacket)
	})
	t.Run("FR13 writer closed", func(t *testing.T) {
		wc, ws := net.Pipe()
		require.NoError(t, ws.Close())
		p := &Protocol{DeviceType: gt500, Imei: imei}
		err := p.ConsumeStream(bufio.NewReader(strings.NewReader(f5)), wc, newFakeStore())
		require.Error(t, err)
		assert.NotErrorIs(t, err, io.EOF)
	})
}

func TestFramingSplitSweep(t *testing.T) { // FR3
	for i := 1; i < len(f1); i++ {
		recs, _ := runPipe(t, gt500, []string{f1[:i], f1[i:]}, 0)
		require.Len(t, recs, 1, "split at byte %d", i)
	}
	// realistic: header arrives, pause, rest arrives
	recs, _ := runPipe(t, gt500, []string{"+RESP:GTFRI,072002,1357902468", f1[len("+RESP:GTFRI,072002,1357902468"):]}, 20*time.Millisecond)
	require.Len(t, recs, 1)
	assert.InDelta(t, 31.222073, recs[0].Position.Latitude, 1e-5)
}

// ---- §15.7 Heartbeat ----

func TestHeartbeat(t *testing.T) {
	cases := []struct{ name, in, wantSack string }{
		{"HB1 doc", f5, "+SACK:GTHBD,070002,11F0$"},
		{"HB2 odd version echoed", "+ACK:GTHBD,0320002,135790246811220,,20100214093254,11F0$", "+SACK:GTHBD,0320002,11F0$"},
		{"HB3 count FFFF", setField(f5, 5, "FFFF"), "+SACK:GTHBD,070002,FFFF$"},
		{"HB3 lowercase count", setField(f5, 5, "11f0"), "+SACK:GTHBD,070002,11f0$"},
		{"HB4 other ack", "+ACK:GTQSS,070002,135790246811220,,0000$+ACK:GTFRI,070002,135790246811220,,0001$", ""},
		{"HB5 gv200", "+ACK:GTHBD,040100,135790246811220,,20100214093254,11F0$", "+SACK:GTHBD,040100,11F0$"},
		{"HB6 two heartbeats", f5 + setField(f5, 5, "11F1"), "+SACK:GTHBD,070002,11F0$+SACK:GTHBD,070002,11F1$"},
		{"malformed heartbeat", "+ACK:GTHBD$", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recs, w := mustStream(t, gt500, c.in)
			assert.Empty(t, recs)
			assert.Equal(t, c.wantSack, w)
		})
	}
}

// ---- §15.5 GT500 + §15.4 field level + §15.9 contract ----

func TestGT500Position(t *testing.T) { // GT1, GP1, DS1-DS6
	recs := parseLine(t, gt500, f1)
	require.Len(t, recs, 1)
	r := recs[0]
	assert.Equal(t, imei, r.Imei)
	assert.Equal(t, gt500, r.DeviceType)
	assert.Equal(t, "RESP:GTFRI", r.MessageType)
	require.NotNil(t, r.Position)
	require.NotNil(t, r.VehicleStatus)
	require.NotNil(t, r.Timestamp)
	assert.InDelta(t, 31.222073, r.Position.Latitude, 1e-5)
	assert.InDelta(t, 121.354335, r.Position.Longitude, 1e-5)
	assert.InDelta(t, 90.0, r.Position.Altitude, 1e-5)
	require.NotNil(t, r.Position.Speed)
	assert.InDelta(t, 4.3, *r.Position.Speed, 1e-5)
	assert.Equal(t, float32(72), r.Position.Course)
	assert.Equal(t, int32(0), r.Position.Satellites)
	require.NotNil(t, r.VehicleStatus.Ignition)
	assert.True(t, *r.VehicleStatus.Ignition)
	assert.Equal(t, int32(80), r.BatteryLevel)
	assert.Equal(t, int32(1000), r.Odometer, "GT500 device ODO (km) mapped since Phase 7")
	assert.Equal(t, utc(2009, 2, 14, 1, 32, 54), r.Timestamp.AsTime())
	assert.Equal(t, strings.TrimSuffix(f1, "$"), string(r.GetQueclinkPacket().GetRawData()))
}

func TestGT500Headers(t *testing.T) {
	for _, name := range []string{"GTFRI", "GTGEO", "GTSPD", "GTSOS", "GTRTL", "GTPNL", "GTPOR", "GTNMR", "GTMSA"} { // GT2
		recs := parseLine(t, gt500, withHeader(f1, "+RESP:"+name))
		require.Len(t, recs, 1, name)
		assert.Equal(t, "RESP:"+name, recs[0].MessageType)
		assert.Equal(t, int32(80), recs[0].BatteryLevel)
	}
	for _, line := range []string{gtStt, gtBpl} { // GT4: state / low-battery reports carry a position too (Phase 8)
		recs := parseLine(t, gt500, line)
		require.Len(t, recs, 1, line)
		assert.Equal(t, int32(0), recs[0].BatteryLevel, "no battery seen yet on this connection")
	}
	for _, line := range []string{ // UN1-3, UN5: no GPS block anywhere
		gtInf, f6, withHeader(f6, "+RESP:GTPFA"), withHeader(f6, "+RESP:GTPDP"),
		"+RESP:GTALL,070002,135790246811220,,gt500,1,1,,,0000,0000,,,30,30,,,1000,0,30,100,5,,,0002$",
		"+RESP:GTGSM,080100,135790246811220,,FRI,0460,0000,18d8,6141,,,,,,,,,,,,,,,,,,,,,,,,,,,20090214093254,11F0$",
		"+RESP:GTXYZ,070002,135790246811220,,1,2,3$",
		"+RESP:GTDAT,070002,135790246811220,,0,hello world,20090214093254,11F0$",
		"+RESP:GTFRI$", "+RESP:$", "+RESP$", "$",
	} {
		assert.Empty(t, parseLine(t, gt500, line), line)
	}
}

func TestGT500Edges(t *testing.T) {
	one := func(t *testing.T, line string) *types.DeviceStatus {
		t.Helper()
		recs := parseLine(t, gt500, line)
		require.Len(t, recs, 1)
		return recs[0]
	}
	t.Run("GT3 doc +BUFF 22 fields", func(t *testing.T) {
		r := one(t, f2)
		assert.Equal(t, "BUFF:GTFRI", r.MessageType)
		assert.Equal(t, int32(0), r.BatteryLevel)
		assert.InDelta(t, 31.222073, r.Position.Latitude, 1e-5)
	})
	t.Run("GT5 battery values", func(t *testing.T) {
		for in, want := range map[string]int32{"100": 100, "0": 0, "": 0, "abc": 0, "80": 80} {
			assert.Equal(t, want, one(t, setField(f1, 21, in)).BatteryLevel, in)
		}
	})
	t.Run("GT6 device name", func(t *testing.T) {
		assert.Equal(t, int32(80), one(t, setField(f1, 3, "mytracker1")).BatteryLevel)
	})
	t.Run("GT7 geo-fence id and type", func(t *testing.T) {
		one(t, withHeader(setField(setField(f1, 4, "3"), 5, "1"), "+RESP:GTGEO"))
	})
	t.Run("GT8 extra trailing field", func(t *testing.T) {
		r := one(t, insertAfter(f1, 22, ""))
		assert.Equal(t, int32(0), r.BatteryLevel, "battery is only trusted when the length matches")
	})
	t.Run("GT9 multi-point", func(t *testing.T) {
		recs := parseLine(t, gt500, f7)
		require.Len(t, recs, 2)
		assert.Equal(t, utc(2009, 2, 14, 1, 32, 54), recs[0].Timestamp.AsTime())
		assert.Equal(t, utc(2009, 2, 14, 1, 33, 54), recs[1].Timestamp.AsTime())
		assert.Equal(t, int32(80), recs[0].BatteryLevel)
		assert.Equal(t, int32(80), recs[1].BatteryLevel)
	})
	t.Run("GP6 speed and ignition threshold", func(t *testing.T) {
		for in, want := range map[string]struct {
			speed float32
			ign   bool
		}{"": {0, false}, "-1": {0, false}, "3.0": {3, false}, "3.1": {3.1, true}, "4.3": {4.3, true}, "999.9": {999.9, true}} {
			r := one(t, setField(f1, 8, in))
			assert.InDelta(t, want.speed, *r.Position.Speed, 1e-4, in)
			assert.Equal(t, want.ign, *r.VehicleStatus.Ignition, in)
		}
	})
	t.Run("GP7 azimuth and altitude", func(t *testing.T) {
		assert.Equal(t, float32(0), one(t, setField(f1, 9, "")).Position.Course)
		assert.Equal(t, float32(359), one(t, setField(f1, 9, "359")).Position.Course)
		assert.Equal(t, float32(0), one(t, setField(f1, 10, "")).Position.Altitude)
		assert.InDelta(t, -12.3, one(t, setField(f1, 10, "-12.3")).Position.Altitude, 1e-5)
	})
	t.Run("GP8 time", func(t *testing.T) {
		assert.Empty(t, parseLine(t, gt500, setField(f1, 13, "2009021401325")), "13-char time")
		assert.Empty(t, parseLine(t, gt500, setField(f1, 13, "")), "empty time")
		assert.Equal(t, utc(2024, 12, 31, 23, 59, 59), one(t, setField(f1, 13, "20241231235959")).Timestamp.AsTime())
	})
	t.Run("GP5 OWH empty lon/lat/time", func(t *testing.T) {
		assert.Empty(t, parseLine(t, gt500, setField(setField(setField(f1, 11, ""), 12, ""), 13, "")))
		assert.Empty(t, parseLine(t, gt500, setField(f1, 11, "")), "empty lon only")
		assert.Empty(t, parseLine(t, gt500, setField(f1, 12, "")), "empty lat only")
	})
	t.Run("GP12 Number 0", func(t *testing.T) {
		assert.Empty(t, parseLine(t, gt500, setField(f1, 6, "0")))
		assert.Empty(t, parseLine(t, gt500, setField(f1, 6, "00")))
	})
	t.Run("GP13 Number 15 and 16", func(t *testing.T) {
		block := "1,4.3,72,90.0,121.354335,31.222073,%s,0460,0000,18d8,6141"
		var blocks []string
		for i := 0; i < 15; i++ {
			blocks = append(blocks, fmt.Sprintf(block, fmt.Sprintf("200902140132%02d", i)))
		}
		body := "+RESP:GTFRI,072002,135790246811220,,0,0,%d," + strings.Join(blocks, ",") + ",000ce7000000,-56,1000,80,20090214093254,11F0$"
		recs := parseLine(t, gt500, fmt.Sprintf(body, 15))
		require.Len(t, recs, 15)
		assert.Equal(t, int32(80), recs[14].BatteryLevel)
		recs = parseLine(t, gt500, fmt.Sprintf(body, 16))
		assert.Len(t, recs, 15, "Number=16 with 15 blocks: parse what fits")
	})
	t.Run("GP14 Number 2 but one block", func(t *testing.T) {
		r := one(t, setField(f1, 6, "2"))
		assert.Equal(t, int32(80), r.BatteryLevel)
	})
	t.Run("GP15 spaces inside fields", func(t *testing.T) {
		assert.InDelta(t, 4.3, *one(t, setField(f1, 8, " 4.3 ")).Position.Speed, 1e-5)
	})
	t.Run("GP16 non-numeric coordinates", func(t *testing.T) {
		assert.Empty(t, parseLine(t, gt500, setField(f1, 12, "abc")))
		assert.Empty(t, parseLine(t, gt500, setField(f1, 11, "1.2.3")))
	})
	t.Run("GP4 zero coordinates", func(t *testing.T) {
		assert.Empty(t, parseLine(t, gt500, setField(setField(f1, 11, "0"), 12, "0")))
		assert.Empty(t, parseLine(t, gt500, setField(setField(f1, 11, "0.0"), 12, "0.000000")))
		one(t, setField(f1, 11, "0")) // lon 0 with a real lat is a valid point (Greenwich meridian)
	})
	t.Run("wrong device type", func(t *testing.T) {
		assert.Empty(t, (&Protocol{DeviceType: types.DeviceType_TELTONIKA, Imei: imei}).parseReport(splitFields(strings.TrimSuffix(f1, "$")), nil))
	})
}

// ---- §15.6 GV200 ----

func TestGV200(t *testing.T) {
	one := func(t *testing.T, line string) *types.DeviceStatus {
		t.Helper()
		recs := parseLine(t, gv200, line)
		require.Len(t, recs, 1)
		return recs[0]
	}
	t.Run("GV1 doc GTFRI", func(t *testing.T) {
		r := one(t, gv1)
		assert.Equal(t, gv200, r.DeviceType)
		assert.Equal(t, "RESP:GTFRI", r.MessageType)
		assert.Equal(t, int32(2000), r.Odometer)
		assert.Equal(t, int32(0), r.BatteryLevel)
		assert.Equal(t, utc(2009, 2, 14, 1, 32, 54), r.Timestamp.AsTime())
		assert.InDelta(t, 31.222073, r.Position.Latitude, 1e-5)
		assert.True(t, *r.VehicleStatus.Ignition)
	})
	t.Run("GV2 doc 2-point GTFRI", func(t *testing.T) {
		recs := parseLine(t, gv200, gv2)
		require.Len(t, recs, 2)
		assert.Equal(t, utc(2009, 2, 14, 1, 32, 54), recs[0].Timestamp.AsTime())
		assert.Equal(t, utc(2009, 1, 1, 0, 0, 0), recs[1].Timestamp.AsTime(), "HDOP 0 with coordinates is still emitted")
		assert.Equal(t, int32(2000), recs[1].Odometer)
	})
	t.Run("GV3 traccar 517 empty tail", func(t *testing.T) {
		r := one(t, gv517)
		assert.Equal(t, int32(0), r.Odometer)
		assert.InDelta(t, 50.798077, r.Position.Latitude, 1e-5)
		assert.InDelta(t, 8.924243, r.Position.Longitude, 1e-5)
		assert.False(t, *r.VehicleStatus.Ignition)
	})
	t.Run("GV4 GP2 GP3 prefix 35, HDOP 0, S/W hemispheres", func(t *testing.T) {
		r := one(t, gvF16)
		assert.InDelta(t, -70.514872, r.Position.Longitude, 1e-5)
		assert.InDelta(t, -33.361021, r.Position.Latitude, 1e-5)
		assert.InDelta(t, 820.8, r.Position.Altitude, 1e-3)
		assert.Equal(t, int32(0), r.Odometer)
	})
	t.Run("GV5 GP4 GP9 no fix", func(t *testing.T) {
		assert.Empty(t, parseLine(t, gv200, gvF17))
	})
	t.Run("GV6 doc GTERI extra field", func(t *testing.T) {
		r := one(t, gvEri)
		assert.Equal(t, "RESP:GTERI", r.MessageType)
		assert.Equal(t, int32(0), r.Odometer)
		assert.InDelta(t, 117.200986, r.Position.Longitude, 1e-5)
	})
	t.Run("GV7 real GTERI", func(t *testing.T) {
		assert.Equal(t, int32(17363), one(t, gvEri2024).Odometer)
		assert.Equal(t, int32(15529), one(t, gvEri2021).Odometer)
		r := one(t, gvBuffEri)
		assert.Equal(t, "BUFF:GTERI", r.MessageType)
		assert.Equal(t, int32(19273), r.Odometer)
		assert.Equal(t, int32(84795), one(t, gvBuffEri2).Odometer)
	})
	t.Run("GV8 event positions", func(t *testing.T) {
		for _, name := range []string{"GTTOW", "GTDIS", "GTIOB", "GTGEO", "GTSPD", "GTSOS", "GTRTL", "GTDOG", "GTIGL", "GTHBM"} {
			r := one(t, withHeader(gvTow, "+RESP:"+name))
			assert.Equal(t, "RESP:"+name, r.MessageType)
			assert.Equal(t, int32(2000), r.Odometer, name)
		}
	})
	t.Run("GV9 GTIGN GTIGF", func(t *testing.T) {
		assert.Equal(t, int32(2000), one(t, gvIgn).Odometer)
		assert.Equal(t, int32(2000), one(t, gvIgf).Odometer)
		assert.Equal(t, "RESP:GTIGF", one(t, gvIgf).MessageType)
	})
	t.Run("GV10 log-only reports", func(t *testing.T) {
		require.Len(t, parseLine(t, gv200, gvStt520), 1, "GTSTT carries a position (Phase 8)")
		for _, line := range []string{gvInf307, gvInf421, f6, "+RESP:GTVER,040100,135790246811220,,GV200,0100,0100,20090214093254,11F0$"} {
			assert.Empty(t, parseLine(t, gv200, line), line)
		}
	})
	t.Run("GP10 GV11 append mask", func(t *testing.T) {
		r := one(t, setField(gv1, 18, "")) // empty mask → width 12
		assert.Equal(t, int32(2000), r.Odometer)
		r = one(t, insertAfter(setField(gv1, 18, "01"), 18, "9")) // bit0 → satellites
		assert.Equal(t, int32(9), r.Position.Satellites)
		assert.Equal(t, int32(2000), r.Odometer)
		r = one(t, insertAfter(setField(gv1, 18, "02"), 18, "0")) // bit1 → trigger type, no satellites
		assert.Equal(t, int32(0), r.Position.Satellites)
		assert.Equal(t, int32(2000), r.Odometer)
		r = one(t, insertAfter(setField(gv1, 18, "03"), 18, "9", "0")) // both
		assert.Equal(t, int32(9), r.Position.Satellites)
		assert.Equal(t, int32(2000), r.Odometer)
	})
	t.Run("GP11 multi-point with append mask", func(t *testing.T) {
		line := insertAfter(setField(gv2, 18, "01"), 18, "7") // block 2 now starts at 20, its mask at 31
		line = insertAfter(setField(line, 31, "01"), 31, "8")
		recs := parseLine(t, gv200, line)
		require.Len(t, recs, 2)
		assert.Equal(t, int32(7), recs[0].Position.Satellites)
		assert.Equal(t, int32(8), recs[1].Position.Satellites)
		assert.Equal(t, int32(2000), recs[1].Odometer)
	})
	t.Run("GV12 ERI mask value irrelevant", func(t *testing.T) {
		assert.Equal(t, int32(17363), one(t, setField(gvEri2024, 4, "00000001")).Odometer)
		assert.Equal(t, int32(17363), one(t, setField(gvEri2024, 4, "00000000")).Odometer)
	})
	t.Run("DS5 odometer bounds", func(t *testing.T) {
		assert.Equal(t, int32(4294967), one(t, setField(gv1, 19, "4294967.0")).Odometer)
		assert.Equal(t, int32(17363), one(t, setField(gv1, 19, "17363.6")).Odometer)
		assert.Equal(t, int32(0), one(t, setField(gv1, 19, "")).Odometer)
		assert.Equal(t, int32(0), one(t, setField(gv1, 19, "-5")).Odometer)
		assert.Equal(t, int32(0), one(t, setField(gv1, 19, "99999999999")).Odometer)
	})
	t.Run("other Queclink models do not break the GV200 parser", func(t *testing.T) {
		r := one(t, f18) // cv100: empty mask, 8-hex cell id, HDOP 0
		assert.InDelta(t, 22.537178, r.Position.Latitude, 1e-5)
		r = one(t, f19) // GV310LAU: mask 01 + satellites 12
		assert.Equal(t, int32(12), r.Position.Satellites)
		assert.Equal(t, int32(358), r.Odometer)
		assert.InDelta(t, -90.544928, r.Position.Longitude, 1e-5)
		r = one(t, f20) // 10-char version, mask 02 + trigger type
		assert.Equal(t, "BUFF:GTFRI", r.MessageType)
		assert.Equal(t, int32(0), r.Position.Satellites)
		assert.InDelta(t, -33.524718, r.Position.Latitude, 1e-5)
		r = one(t, f21) // GV600M
		assert.Equal(t, int32(27), r.Odometer)
		assert.InDelta(t, -122.234955, r.Position.Longitude, 1e-5)
	})
	t.Run("truncated GV200 lines", func(t *testing.T) {
		for _, line := range []string{"+RESP:GTFRI,040100,135790246811220,,,10,1$", "+RESP:GTERI,040100,135790246811220,,00000001,,10,1$", "+RESP:GTIGN,040100,135790246811220,,1200$", "+RESP:GTFRI,040100,135790246811220,,,10,1,1,4.3,92,70.0,121.354335,31.222073$"} {
			assert.Empty(t, parseLine(t, gv200, line), line)
		}
	})
}

// ---- §15.9 contract across every fixture, and fuzz ----

func checkContract(t *testing.T, dt types.DeviceType, line string, recs []*types.DeviceStatus) {
	t.Helper()
	for _, r := range recs {
		require.NotNil(t, r.Position, line)
		require.NotNil(t, r.Position.Speed, line)
		require.NotNil(t, r.VehicleStatus, line)
		require.NotNil(t, r.VehicleStatus.Ignition, line)
		require.NotNil(t, r.Timestamp, line)
		assert.Equal(t, imei, r.Imei)
		assert.Equal(t, dt, r.DeviceType)
		assert.False(t, strings.HasPrefix(r.MessageType, "+"), line)
		assert.Equal(t, strings.TrimLeft(splitFields(line)[0], "+"), r.MessageType, line)
		assert.Equal(t, time.UTC, r.Timestamp.AsTime().Location())
		assert.NotNil(t, r.GetQueclinkPacket(), line)
	}
}

func TestContractOnAllFixtures(t *testing.T) {
	for _, dt := range []types.DeviceType{gt500, gv200} {
		for _, line := range allFixtures {
			checkContract(t, dt, line, parseLine(t, dt, line))
		}
	}
}

func FuzzParseReport(f *testing.F) {
	for _, line := range append(append(append([]string{}, allFixtures...), phase3Fixtures...), phase8Fixtures...) {
		f.Add(line, true)
		f.Add(line, false)
	}
	for _, line := range gl300Fixtures {
		f.Add(line, true)
		f.Add(line, false)
	}
	f.Add(gvIda, true)
	f.Add(strings.Repeat(",", 100)+"$", true)
	f.Add("+RESP:GTFRI,,,,,,999999999,"+strings.Repeat("1,", 30)+"$", true)
	f.Fuzz(func(t *testing.T, line string, isGV200 bool) {
		dt := gt500
		if isGV200 {
			dt = gv200
		}
		raw := strings.TrimSuffix(line, "$")
		recs := (&Protocol{DeviceType: dt, Imei: imei}).parseReport(splitFields(raw), []byte(raw))
		checkContract(t, dt, raw, recs)
	})
}

// ---- Phase 3: GV200 real ignition, alarm flags, remaining GV200 layouts ----

const (
	// GV200 doc
	gvMpn = "+RESP:GTMPN,040100,135790246811220,,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20090214093254,11F0$"
	gvMpf = "+RESP:GTMPF,040100,135790246811220,,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20090214093254,11F0$"
	gvBtc = "+RESP:GTBTC,040100,135790246811220,,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20090214093254,11F0$"
	gvJdr = "+RESP:GTJDR,040408,135790246811220,,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20090214093254,11F0$"
	gvJds = "+RESP:GTJDS,040408,135790246811220,,2,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20090214093254,11F0$"
	gvStc = "+RESP:GTSTC,040100,135790246811220,,,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20090214093254,11F0$"
	gvBpl = "+RESP:GTBPL,040100,135790246811220,,3.53,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20090214093254,11F0$"
	gvAnt = "+RESP:GTANT,040100,135790246811220,,0,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20090214093254,11F0$"
	gvIdn = "+RESP:GTIDN,040100,135790246811220,,,,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	gvIdf = "+RESP:GTIDF,040100,135790246811220,,22,300,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	gvHbm = "+RESP:GTHBM,040100,135790246811220,,,10,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	gvDis = "+RESP:GTDIS,040100,135790246811220,,,20,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	// traccar corpus, other Queclink models sharing the GV200 event layouts
	tDis    = "+RESP:GTDIS,270302,867162025086950,,,21,1,1,0.0,81,117.8,-116.862025,32.453497,20180309084516,0334,0020,2B24,52CA916,00,1286.2,20180309084517,357E$"
	tIglOff = "+RESP:GTIGL,270302,867162025085234,,,01,1,1,0.0,92,111.2,-116.867638,32.450321,20180327070838,0334,0020,2B24,52CC3DE,00,243.1,20180327070839,2A9A$"
	tIglOn  = "+RESP:GTIGL,DF0200,868487004353181,cv100,,00,1,1,0.0,0,264.8,114.015502,22.537327,20210608064027,0460,0001,25F8,061A7D02,,0.0,20210608144025,32D1$"
	tSos    = "+RESP:GTSOS,020102,135790246811220,,0,0,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,,20090214093254,11F0$"
	tDog    = "+RESP:GTDOG,020102,135790246811220,,0,0,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	tMpn    = "+RESP:GTMPN,450102,865084030001323,gb100,0,1.6,0,-93.1,121.393023,31.164105,20170619103113,0460,0000,1806,2142,00,20170619103143,0512$"
	tBpl    = "+RESP:GTBPL,110100,358688000000158,,3.53,0,4.3,92,70.0,121.354335,31.222073,20110214013254,0460,0000,18d8,6141,,20110214093254,001F$"
	tStt21  = "+RESP:GTSTT,060228,862894020178276,,21,0,0.0,0,411.3,-63.169745,-17.776330,20160319132220,0736,0003,6AD4,5BAA,00,20160319092223,1FBD$"
	tIgnBuf = "+BUFF:GTIGN,6E0202,868589060169789,ra79,379,1,0.0,105,532.2,-70.616413,-33.393457,20240610201712,0730,0001,333A,00CFA301,01,11,,0.0,20240610201713,3AE2$"
	tHbmCv  = "+RESP:GTHBM,BD0214,869409069481243,cv100,,12,1,1,5.6,344,274.7,-1.601826,6.666228,20250114094616,0620,0001,3EE8,00016E04,,,20250114094616,20250114094616,2862$"
)

var phase3Fixtures = []string{gvMpn, gvMpf, gvBtc, gvJdr, gvJds, gvStc, gvBpl, gvAnt, gvIdn, gvIdf, gvHbm, gvDis,
	tDis, tIglOff, tIglOn, tSos, tDog, tMpn, tBpl, tStt21, tIgnBuf, tHbmCv}

func TestGV200MoreLayouts(t *testing.T) {
	cases := []struct {
		name, line string
		wantOdo    int32
		wantLon    float64
	}{
		{"GTMPN 18 fields", gvMpn, 0, 121.354335},
		{"GTMPF", gvMpf, 0, 121.354335},
		{"GTBTC", gvBtc, 0, 121.354335},
		{"GTJDR", gvJdr, 0, 121.354335},
		{"GTJDS 19 fields", gvJds, 0, 121.354335},
		{"GTSTC", gvStc, 0, 121.354335},
		{"GTBPL", gvBpl, 0, 121.354335},
		{"GTANT", gvAnt, 0, 121.354335},
		{"GTIDN 21 fields", gvIdn, 2000, 121.354335},
		{"GTIDF", gvIdf, 2000, 121.354335},
		{"GTDIS doc", gvDis, 2000, 121.354335},
		{"real GTDIS", tDis, 1286, -116.862025},
		{"real GTIGL", tIglOff, 243, -116.867638},
		{"real GTSOS empty mileage", tSos, 0, 121.354335},
		{"real GTDOG", tDog, 2000, 121.354335},
		{"real GTMPN gb100", tMpn, 0, 121.393023},
		{"real GTBPL empty mask", tBpl, 0, 121.354335},
		{"real +BUFF:GTIGN append mask", tIgnBuf, 0, -70.616413},
		{"real GTHBM extra field", tHbmCv, 0, -1.601826},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recs := parseLine(t, gv200, c.line)
			require.Len(t, recs, 1)
			assert.Equal(t, c.wantOdo, recs[0].Odometer)
			assert.InDelta(t, c.wantLon, recs[0].Position.Longitude, 1e-5)
			assert.Equal(t, strings.TrimPrefix(splitFields(c.line)[0], "+"), recs[0].MessageType)
		})
	}
	assert.Equal(t, int32(11), parseLine(t, gv200, tIgnBuf)[0].Position.Satellites)
	recs := parseLine(t, gv200, tStt21)
	require.Len(t, recs, 1, "GTSTT carries a position (Phase 8)")
	assert.True(t, *recs[0].VehicleStatus.Ignition, "state 21 = ignition on")
	assert.Equal(t, int32(0), recs[0].Odometer)
}

func TestGV200Alarms(t *testing.T) {
	type want struct{ tow, brake, accel, rash, spd, unplug, idle, sos, in, out, inputs bool }
	cases := []struct {
		name, line string
		want       want
	}{
		{"GTTOW", gvTow, want{tow: true}},
		{"GTHBM 10 braking", gvHbm, want{brake: true, rash: true}},
		{"GTHBM 11 acceleration", setField(gvHbm, 5, "11"), want{accel: true, rash: true}},
		{"GTHBM 31 acceleration at high speed", setField(gvHbm, 5, "31"), want{accel: true, rash: true}},
		{"GTHBM 12 unknown type", tHbmCv, want{rash: true}},
		{"GTSPD 00 outside range", withHeader(gvTow, "+RESP:GTSPD"), want{spd: true}},
		{"GTSPD 01 inside range", withHeader(setField(gvTow, 5, "01"), "+RESP:GTSPD"), want{spd: true}},
		{"GTMPF", gvMpf, want{unplug: true}},
		{"GTIDN", gvIdn, want{idle: true}},
		{"GTSOS 10 input 1", withHeader(setField(gvTow, 5, "10"), "+RESP:GTSOS"), want{sos: true}},
		{"real GTSOS", tSos, want{sos: true}},
		{"GTGEO 01 enter fence 0", withHeader(setField(gvTow, 5, "01"), "+RESP:GTGEO"), want{in: true}},
		{"GTGEO 30 exit fence 3", withHeader(setField(gvTow, 5, "30"), "+RESP:GTGEO"), want{out: true}},
		{"GTDIS 20 input 2 off", gvDis, want{inputs: true}},
		{"real GTDIS 21", tDis, want{inputs: true}},
		{"GTFRI no flags", gv1, want{}},
		{"GTERI no flags", gvEri2024, want{}},
		{"GTIGN no flags", gvIgn, want{}},
		{"GTIGL no flags", tIglOff, want{}},
		{"GTMPN no flags", gvMpn, want{}},
		{"GTIDF no flags", gvIdf, want{}},
		{"GTJDR/GTJDS/GTBTC/GTSTC/GTBPL/GTANT/GTDOG/GTIOB/GTRTL no flags", gvJds, want{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recs := parseLine(t, gv200, c.line)
			require.Len(t, recs, 1)
			vs := recs[0].VehicleStatus
			got := want{vs.Towing, vs.HarshBraking, vs.HarshAcceleration, vs.RashDriving, vs.OverSpeeding, vs.UnplugBattery,
				vs.ExcessiveIdling, vs.SosButtonPressed, vs.EntringGeofence, vs.ExitingGeofence, vs.InputsTriggering}
			assert.Equal(t, c.want, got)
			assert.False(t, vs.CrashDetection || vs.AutoGeofence || vs.HarshCornering || vs.FatigueDriving)
		})
	}
	for _, line := range []string{gvJdr, gvBtc, gvStc, gvBpl, gvAnt, gvJds, tBpl, tMpn} {
		recs := parseLine(t, gv200, line)
		require.Len(t, recs, 1, line)
		assert.Equal(t, &types.VehicleStatus{Ignition: recs[0].VehicleStatus.Ignition}, recs[0].VehicleStatus, line)
	}
	for _, h := range []string{"GTDOG", "GTIOB", "GTRTL"} {
		recs := parseLine(t, gv200, withHeader(gvTow, "+RESP:"+h))
		require.Len(t, recs, 1, h)
		assert.Equal(t, &types.VehicleStatus{Ignition: recs[0].VehicleStatus.Ignition}, recs[0].VehicleStatus, h)
	}
}

func TestGT500Alarms(t *testing.T) {
	cases := []struct {
		name, line        string
		sos, spd, in, out bool
	}{
		{"GTSOS", withHeader(f1, "+RESP:GTSOS"), true, false, false, false},
		{"GTSPD type 1", withHeader(setField(f1, 5, "1"), "+RESP:GTSPD"), false, true, false, false},
		{"GTSPD type 0", withHeader(f1, "+RESP:GTSPD"), false, true, false, false},
		{"GTGEO fence 2 enter", withHeader(setField(setField(f1, 4, "2"), 5, "1"), "+RESP:GTGEO"), false, false, true, false},
		{"GTGEO fence 2 exit", withHeader(setField(f1, 4, "2"), "+RESP:GTGEO"), false, false, false, true},
		{"GTMSA fall: no matching flag", withHeader(f1, "+RESP:GTMSA"), false, false, false, false},
		{"GTNMR rest->motion: no flag", withHeader(setField(f1, 5, "1"), "+RESP:GTNMR"), false, false, false, false},
		{"GTFRI", f1, false, false, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recs := parseLine(t, gt500, c.line)
			require.Len(t, recs, 1)
			vs := recs[0].VehicleStatus
			assert.Equal(t, []bool{c.sos, c.spd, c.in, c.out}, []bool{vs.SosButtonPressed, vs.OverSpeeding, vs.EntringGeofence, vs.ExitingGeofence})
			assert.False(t, vs.Towing || vs.RashDriving || vs.UnplugBattery || vs.ExcessiveIdling || vs.InputsTriggering)
		})
	}
	recs := parseLine(t, gt500, gtBpl)
	require.Len(t, recs, 1, "GT500 GTBPL is a position (Phase 8); its voltage is not a percentage")
	assert.Equal(t, &types.VehicleStatus{Ignition: recs[0].VehicleStatus.Ignition}, recs[0].VehicleStatus)
}

// TestGV200IgnitionState: the real signal replaces the speed rule for the rest of the connection.
func TestGV200IgnitionState(t *testing.T) {
	fast := gv1                     // speed 4.3 → speed rule says on
	slow := setField(gv1, 8, "0.0") // speed 0 → speed rule says off
	ign := func(recs []*types.DeviceStatus) []bool {
		var out []bool
		for _, r := range recs {
			out = append(out, *r.VehicleStatus.Ignition)
		}
		return out
	}
	cases := []struct {
		name  string
		lines []string
		want  []bool
	}{
		{"speed rule until a signal arrives", []string{fast, slow}, []bool{true, false}},
		{"GTIGN then slow position", []string{gvIgn, slow}, []bool{true, true}},
		{"GTIGF then fast position", []string{gvIgf, fast}, []bool{false, false}},
		{"GTIGL 00 = on", []string{withHeader(setField(gvTow, 5, "00"), "+RESP:GTIGL"), slow}, []bool{true, true}},
		{"GTIGL 01 = off", []string{tIglOff, fast}, []bool{false, false}},
		{"real GTIGL 00 cv100", []string{tIglOn, slow}, []bool{true, true}},
		// GTSTT lines are positions too (Phase 8): the state applies to the GTSTT record itself
		{"GTSTT 11 off", []string{gvIgn, setField(gvStt520, 4, "11"), fast}, []bool{true, false, false}},
		{"GTSTT 12 off", []string{gvIgn, setField(gvStt520, 4, "12"), fast}, []bool{true, false, false}},
		{"GTSTT 21 on", []string{tStt21, slow}, []bool{true, true}},
		{"GTSTT 22 on", []string{setField(gvStt520, 4, "22"), slow}, []bool{true, true}},
		{"GTSTT 42 leaves state alone", []string{gvIgf, gvStt520, fast}, []bool{false, false, false}},
		{"GTSTT 41 without prior signal keeps speed rule", []string{setField(gvStt520, 4, "41"), fast, slow}, []bool{false, true, false}},
		{"GTSTT 16 tow leaves state alone", []string{gvIgn, setField(gvStt520, 4, "16"), slow}, []bool{true, true, true}},
		{"GTINF 41 leaves state alone", []string{gvIgf, gvInf307, fast}, []bool{false, false}},
		{"GTINF 21 on", []string{setField(gvInf307, 4, "21"), slow}, []bool{true}},
		{"GTINF 16 doc example leaves state alone", []string{gvInf421, fast}, []bool{true}},
		{"on, off, on sequence", []string{gvIgn, slow, gvIgf, fast, gvIgn, slow}, []bool{true, true, false, false, true, true}},
		{"+BUFF GTIGN counts", []string{tIgnBuf, slow}, []bool{true, true}},
		{"GTIGF record itself is off despite speed 4.3", []string{gvIgf}, []bool{false}},
		{"GTIGN record itself is on", []string{setField(gvIgn, 6, "0.0")}, []bool{true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recs, _ := mustStream(t, gv200, strings.Join(c.lines, ""))
			assert.Equal(t, c.want, ign(recs))
		})
	}
	t.Run("state is per connection", func(t *testing.T) {
		recs, _ := mustStream(t, gv200, gvIgf+fast)
		assert.Equal(t, []bool{false, false}, ign(recs))
		recs, _ = mustStream(t, gv200, fast) // new Protocol → speed rule again
		assert.Equal(t, []bool{true}, ign(recs))
	})
	t.Run("GV200-only ignition reports under a GT500 registration", func(t *testing.T) {
		recs, _ := mustStream(t, gt500, gvIgf+gvIgn+f1) // positions kept via the field scan; GTIGN/GTIGF never touch GT500 state
		assert.Equal(t, []bool{true, true, true}, ign(recs), "speed rule (4.3 km/h) on all three")
	})
}

// TestGT500MotionSensor: the GT500 has no ignition line; its motion sensor (GTSTT 41/42, GTNMR 0/1)
// is the movement signal, GPS speed only until the first state arrives or after state 99.
func TestGT500MotionSensor(t *testing.T) {
	fast := f1                   // 4.3 km/h → speed rule says on
	slow := setField(f1, 8, "0") // 0 km/h → speed rule says off
	stt41, stt42 := gtStt, setField(gtStt, 4, "42")
	nmrRest := withHeader(f1, "+RESP:GTNMR")                   // report type 0 = motion → rest
	nmrMove := withHeader(setField(f1, 5, "1"), "+RESP:GTNMR") // report type 1 = rest → motion
	inf41, inf99 := gtInf, setField(gtInf, 4, "99")
	ign := func(recs []*types.DeviceStatus) []bool {
		var out []bool
		for _, r := range recs {
			out = append(out, *r.VehicleStatus.Ignition)
		}
		return out
	}
	cases := []struct {
		name  string
		lines []string
		want  []bool
	}{
		{"speed rule before any state", []string{fast, slow}, []bool{true, false}},
		{"GTSTT 41 still: fast GPS jitter is not movement", []string{stt41, fast}, []bool{false, false}},
		{"GTSTT 42 moving: GPS speed 0 is still moving", []string{stt42, slow}, []bool{true, true}},
		{"GTNMR rest→motion record itself is on, and sticks", []string{nmrMove, slow}, []bool{true, true}},
		{"GTNMR motion→rest record itself is off, and sticks", []string{nmrRest, fast}, []bool{false, false}},
		{"GTINF 41 sets still", []string{inf41, fast}, []bool{false}},
		{"GTINF 99 motion detection off → back to speed rule", []string{stt42, slow, inf99, slow, fast}, []bool{true, true, false, true}},
		{"full walk: still, move, still", []string{stt41, slow, stt42, slow, fast, stt41, fast}, []bool{false, false, true, true, true, false, false}},
		{"GTNMR and GTSTT agree in sequence", []string{nmrMove, stt41, fast, nmrMove, slow}, []bool{true, false, false, true, true}},
		{"GV200 ignition reports do not touch GT500 state", []string{stt42, gvIgf, slow}, []bool{true, true, true}}, // gvIgf: scanned position, state stays 42
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recs, _ := mustStream(t, gt500, strings.Join(c.lines, ""))
			assert.Equal(t, c.want, ign(recs))
		})
	}
	t.Run("GT500 state is per connection", func(t *testing.T) {
		recs, _ := mustStream(t, gt500, stt42+slow)
		assert.Equal(t, []bool{true, true}, ign(recs))
		recs, _ = mustStream(t, gt500, slow)
		assert.Equal(t, []bool{false}, ign(recs))
	})
	t.Run("GV200 keeps ignoring 41/42 and 99", func(t *testing.T) {
		recs, _ := mustStream(t, gv200, setField(gvStt520, 4, "41")+gv1+setField(gvStt520, 4, "42")+setField(gv1, 8, "0.0")+setField(gvInf307, 4, "99")+gv1)
		assert.Equal(t, []bool{false, true, false, false, true}, ign(recs), "GV200 without a wired ignition stays on the speed rule (GTSTT records at 0.0 km/h are off)")
		recs, _ = mustStream(t, gv200, gvIgn+setField(gvInf307, 4, "99")+setField(gv1, 8, "0.0"))
		assert.Equal(t, []bool{true, true}, ign(recs), "99 never clears a real GV200 ignition signal")
	})
}

func TestContractOnPhase3Fixtures(t *testing.T) {
	for _, dt := range []types.DeviceType{gt500, gv200} {
		for _, line := range phase3Fixtures {
			checkContract(t, dt, line, parseLine(t, dt, line))
		}
	}
}

// ---- Phase 4: HEX-mode detector, ignored-header counter, generic SACK switch ----

func TestLoginRejectsHexMode(t *testing.T) { // T21 / UN4: same device in Protocol Format 1
	for _, in := range []string{
		"+RSP\x0b\xff\xff\xff\xbf\x00\x5c\x45\x01\x01\x01\x08\x56\x32\x54\x03",
		"+EVT\x0c\x00\xfc\x1f\xbf\x00\x5c\x45\x01\x01\x01\x08\x56\x32\x54\x03",
		"+HBD\x00\x00\x00\x00\x00\x00\x00\x00",
		"+ACK\x00\x00\x00\x00\x00\x00\x00\x00", // binary ACK: no ':' at [4]
		"+LGN\x00\xff\x00\x26\xfe\x11\x0b\x07\x02\x01\x06\x56\x34\x54",
		"+\x00\x00\x38\x00\x08\x65\x13\x40\x50\x94\x72\x26\x82",
		"+ACK:X,1,2$", // ASCII but not a Queclink ack
	} {
		p := &Protocol{}
		_, _, err := p.Login(bufio.NewReader(strings.NewReader(in)))
		assert.ErrorIs(t, err, errs.ErrUnknownProtocol, "%q", in)
		assert.Empty(t, p.GetDeviceID())
	}
}

func TestIgnoredHeadersAreCounted(t *testing.T) { // T22
	p := &Protocol{DeviceType: gt500, Imei: imei}
	st := newFakeStore()
	in := f1 + gtStt + gtBpl + gtBpl + "+RESP:GTXYZ,1,2$" + "+ACK:GTQSS,070002,135790246811220,,0000$" + "garbage$" + f1
	require.ErrorIs(t, p.ConsumeStream(bufio.NewReader(strings.NewReader(in)), io.Discard, st), io.EOF)
	assert.Len(t, st.drain(), 5, "f1, GTSTT, 2×GTBPL, f1 are all positions")
	assert.Equal(t, map[string]int{"+RESP:GTXYZ": 1, "garbage": 1}, p.ignored,
		"+ACK:GTQSS is a command response, not ignored; positions are not counted; GTXYZ has no GPS block to scan")
	assert.Len(t, st.resp, 1, "+ACK:GTQSS forwarded as a DeviceResponse")
	gv := &Protocol{DeviceType: gv200, Imei: imei}
	require.ErrorIs(t, gv.ConsumeStream(bufio.NewReader(strings.NewReader(gv1+gvInf307+"+RESP:GTVER,040100,135790246811220,,GV200,0100,0100,20090214093254,11F0$")), io.Discard, newFakeStore()), io.EOF)
	assert.Equal(t, map[string]int{"+RESP:GTVER": 1}, gv.ignored)
}

func TestGenericSackDisabled(t *testing.T) { // T23: only heartbeats are answered while sendGenericSack is false
	assert.False(t, sendGenericSack)
	_, w := mustStream(t, gt500, f1+f5+gv1)
	assert.Equal(t, "+SACK:GTHBD,070002,11F0$", w)
}

// ---- Phase 5: server → device commands ----

func TestSendCommandToDevice(t *testing.T) {
	gv := &Protocol{DeviceType: gv200, Imei: imei}
	var w bytes.Buffer
	require.NoError(t, gv.SendCommandToDevice(&w, "ignition_off"))
	assert.Regexp(t, `^AT\+GTOUT=gv200,1,0,0,0,0,0,0,0,0,0,0,0,,,,,[0-9A-F]{4}\$$`, w.String(), "immobiliser output 1 energised")
	w.Reset()
	require.NoError(t, gv.SendCommandToDevice(&w, "ignition_on"))
	assert.Regexp(t, `^AT\+GTOUT=gv200,0,0,0,0,0,0,0,0,0,0,0,0,,,,,[0-9A-F]{4}\$$`, w.String(), "released")
	w.Reset()
	require.NoError(t, gv.SendCommandToDevice(&w, "AT+GTRTO=gv200,8,,,,,,0004$"))
	assert.Equal(t, "AT+GTRTO=gv200,8,,,,,,0004$", w.String(), "custom command verbatim")
	w.Reset()
	require.NoError(t, gv.SendCommandToDevice(&w, " AT+GTRTO=gv200,8,,,,,,0004 "))
	assert.Equal(t, "AT+GTRTO=gv200,8,,,,,,0004$", w.String(), "missing $ added, whitespace trimmed")
	assert.Error(t, gv.SendCommandToDevice(&w, ""))

	gt := &Protocol{DeviceType: gt500, Imei: imei}
	w.Reset()
	assert.Error(t, gt.SendCommandToDevice(&w, "ignition_off"), "GT500 has no outputs")
	assert.Empty(t, w.String())
	require.NoError(t, gt.SendCommandToDevice(&w, "AT+GTRTO=gt500,0,,,,,,000B$"))
	assert.Equal(t, "AT+GTRTO=gt500,0,,,,,,000B$", w.String())

	// serials differ between two consecutive presets
	var a, b bytes.Buffer
	require.NoError(t, gv.SendCommandToDevice(&a, "ignition_off"))
	require.NoError(t, gv.SendCommandToDevice(&b, "ignition_off"))
	assert.NotEqual(t, a.String(), b.String())
}

func TestCommandAcksBecomeDeviceResponses(t *testing.T) {
	p := &Protocol{DeviceType: gv200, Imei: imei}
	st := newFakeStore()
	var w bytes.Buffer
	in := "+ACK:GTOUT,040100,135790246811220,,0005,20090214093254,11F0$" +
		"+ACK:GTHBD,040100,135790246811220,,20100214093254,11F1$" + // heartbeat: SACK, not a response
		"+ACK:GTRTO,040100,135790246811220,,000F,20090214093254,11F2$" +
		gv1
	require.ErrorIs(t, p.ConsumeStream(bufio.NewReader(strings.NewReader(in)), &w, st), io.EOF)
	assert.Len(t, st.drain(), 1)
	assert.Equal(t, "+SACK:GTHBD,040100,11F1$", w.String())
	var got []string
	for len(st.resp) > 0 {
		r := <-st.resp
		assert.Equal(t, imei, r.Imei)
		got = append(got, r.Response)
	}
	assert.Equal(t, []string{
		"+ACK:GTOUT,040100,135790246811220,,0005,20090214093254,11F0",
		"+ACK:GTRTO,040100,135790246811220,,000F,20090214093254,11F2",
	}, got)
	assert.Empty(t, p.ignored, "acks are responses, not ignored headers")
}

// ---- Phase 6: GTERI 1-wire temperature ----

func TestGTERITemperature(t *testing.T) {
	one := func(t *testing.T, line string) *types.DeviceStatus {
		t.Helper()
		recs := parseLine(t, gv200, line)
		require.Len(t, recs, 1)
		return recs[0]
	}
	// real 2021 line: two DS18B20-style sensors, first FFE2 = -30 × 0.0625
	assert.InDelta(t, -1.875, one(t, gvEri2021).Temperature, 1e-6)
	assert.Equal(t, int32(15529), one(t, gvEri2021).Odometer)
	// positive reading, single sensor: 01A0 = 416 × 0.0625 = 26 °C
	base := "+RESP:GTERI,040A00,862894022579562,gv200,00000002,,10,1,1,96.1,180,749.7,39.222692,24.165463,20210225065756,0420,0004,759C,3360,00,15529.8,,,2789,,01,00,2,1,282BD47A0B000063,1,01A0,20210225065800,6974$"
	assert.InDelta(t, 26.0, one(t, base).Temperature, 1e-6)
	// 8-hex data as the table allows
	assert.InDelta(t, 26.0, one(t, setField(base, 31, "000001A0")).Temperature, 1e-6)
	// mask bit0+bit1: fuel field shifts the 1-wire block by one
	withFuel := "+RESP:GTERI,040A00,862894022579562,gv200,00000003,,10,1,1,96.1,180,749.7,39.222692,24.165463,20210225065756,0420,0004,759C,3360,00,15529.8,,,2789,,01,00,1,0FA0,1,282BD47A0B000063,1,01A0,20210225065800,6974$"
	assert.InDelta(t, 26.0, one(t, withFuel).Temperature, 1e-6)
	// first sensor unreadable → fall through to the second (FFC8 = -56 × 0.0625)
	assert.InDelta(t, -3.5, one(t, setField(gvEri2021, 31, "zz")).Temperature, 1e-6)
	// first device is not a temperature sensor → use the next one
	twoTypes := setField(setField(gvEri2021, 30, "2"), 34, "0190") // dev1 type 2, dev2 0190 = 25 °C
	assert.InDelta(t, 25.0, one(t, twoTypes).Temperature, 1e-6)
	// no sensors / mask without bit1 / count announced but fields missing / unparseable data → 0, still a record
	for name, line := range map[string]string{
		"count 0":            gvEri2024,
		"mask 00000000":      setField(gvEri2021, 4, "00000000"),
		"mask bit0 only doc": gvEri,
		"truncated block":    strings.Replace(gvEri2021, ",281FDD5D0B000057,1,FFC8", "", 1),
		"both sensors bad":   setField(setField(gvEri2021, 31, "zz"), 34, "0x"),
		"empty data":         setField(setField(gvEri2021, 31, ""), 34, ""),
	} {
		assert.Equal(t, float32(0), one(t, line).Temperature, name)
	}
	// GTFRI never carries a temperature; GT500 never does
	assert.Equal(t, float32(0), one(t, gv1).Temperature)
	assert.Equal(t, float32(0), parseLine(t, gt500, f1)[0].Temperature)
	// multi-point GTERI: the tail temperature is shared by every point
	two := "+RESP:GTERI,040A00,862894022579562,gv200,00000002,,10,2,1,96.1,180,749.7,39.222692,24.165463,20210225065756,0420,0004,759C,3360,00,1,96.1,180,749.7,39.222692,24.165463,20210225065856,0420,0004,759C,3360,00,15529.8,,,2789,,01,00,2,1,282BD47A0B000063,1,FFE2,20210225065900,6975$"
	recs := parseLine(t, gv200, two)
	require.Len(t, recs, 2)
	assert.InDelta(t, -1.875, recs[0].Temperature, 1e-6)
	assert.InDelta(t, -1.875, recs[1].Temperature, 1e-6)
}

// ---- Phase 7: consume the rest — GSM level, GT500 odometer, external voltage, driver ID ----

const gvIda = "+RESP:GTIDA,06020A,862170013895931,,,D2C4FBC5,1,1,1,0.8,0,22.2,117.198630,31.845229,20120802121626,0460,0000,5663,2BB9,00,0.0,,,,,20120802121627,008E$" // GV200 PDF §3.3.1

func TestCsqLevel(t *testing.T) {
	for csq, want := range map[string]int32{"0": 0, "1": 1, "8": 1, "9": 2, "13": 2, "14": 3, "16": 3, "18": 3, "19": 4, "23": 4, "24": 5, "31": 5} {
		got, ok := csqLevel(csq)
		assert.True(t, ok, csq)
		assert.Equal(t, want, got, csq)
	}
	for _, csq := range []string{"", "99", "32", "-1", "abc"} {
		_, ok := csqLevel(csq)
		assert.False(t, ok, csq)
	}
}

func TestGsmFromGTINF(t *testing.T) {
	recs, _ := mustStream(t, gt500, f1+gtInf+f1) // GT500 GTINF CSQ 16 → 3
	require.Len(t, recs, 2)
	assert.Equal(t, int32(-1), recs[0].GsmNetwork, "unknown until the first GTINF")
	assert.Equal(t, int32(3), recs[1].GsmNetwork)
	recs, _ = mustStream(t, gv200, gvInf307+gv1+setField(gvInf307, 6, "99")+gv1+setField(gvInf307, 6, "0")+gv1) // CSQ 24 → 5; 99 keeps last; 0 → no network
	require.Len(t, recs, 3)
	assert.Equal(t, []int32{5, 5, 0}, []int32{recs[0].GsmNetwork, recs[1].GsmNetwork, recs[2].GsmNetwork})
	recs, _ = mustStream(t, gv200, gvInf421+gv1) // doc GTINF CSQ 16 → 3
	assert.Equal(t, int32(3), recs[0].GsmNetwork)
	assert.Equal(t, int32(-1), parseLine(t, gt500, f1)[0].GsmNetwork, "Wi-Fi AP RSSI in GT500 positions is not GSM")
}

func TestGT500Odometer(t *testing.T) {
	assert.Equal(t, int32(1000), parseLine(t, gt500, f1)[0].Odometer)
	assert.Equal(t, int32(80), parseLine(t, gt500, f1)[0].BatteryLevel)
	assert.Equal(t, int32(12345), parseLine(t, gt500, setField(f1, 20, "12345.6"))[0].Odometer)
	assert.Equal(t, int32(0), parseLine(t, gt500, setField(f1, 20, ""))[0].Odometer, "ODO disabled")
	assert.Equal(t, int32(0), parseLine(t, gt500, f2)[0].Odometer, "inconsistent tail: nothing trusted")
	recs := parseLine(t, gt500, f7)
	require.Len(t, recs, 2)
	assert.Equal(t, int32(1000), recs[1].Odometer)
}

func TestGV200ExternalVoltage(t *testing.T) {
	assert.InDelta(t, 12.372, parseLine(t, gv200, gvF16)[0].ControlModuleVoltage, 1e-6)
	assert.InDelta(t, 14.051, parseLine(t, gv200, f18)[0].ControlModuleVoltage, 1e-6)
	assert.Equal(t, float32(0), parseLine(t, gv200, gv1)[0].ControlModuleVoltage, "empty field")
	assert.Equal(t, float32(0), parseLine(t, gv200, gvEri2024)[0].ControlModuleVoltage, "GTERI empty VCC at #5")
	assert.Equal(t, float32(0), parseLine(t, gv200, f21)[0].ControlModuleVoltage, "GTERI VCC 0")
	assert.InDelta(t, 12.194, parseLine(t, gv200, f20)[0].ControlModuleVoltage, 1e-6)
	assert.Equal(t, float32(0), parseLine(t, gv200, gvTow)[0].ControlModuleVoltage, "event reports carry no VCC")
	assert.Equal(t, float32(0), parseLine(t, gt500, f1)[0].ControlModuleVoltage)
}

func TestGTIDADriverID(t *testing.T) {
	recs := parseLine(t, gv200, gvIda)
	require.Len(t, recs, 1)
	r := recs[0]
	assert.Equal(t, "RESP:GTIDA", r.MessageType)
	assert.Equal(t, "D2C4FBC5", r.IdentificationId)
	assert.InDelta(t, 31.845229, r.Position.Latitude, 1e-5)
	assert.Equal(t, int32(0), r.Odometer)
	assert.Equal(t, &types.VehicleStatus{Ignition: r.VehicleStatus.Ignition}, r.VehicleStatus, "no alarm flag")
	assert.Equal(t, "D2C4FBC5", parseLine(t, gv200, setField(gvIda, 6, "0"))[0].IdentificationId, "unauthorised IDs are still reported")
	assert.Empty(t, parseLine(t, gv200, gv1)[0].IdentificationId)
	assert.Empty(t, parseLine(t, gt500, f1)[0].IdentificationId)
	checkContract(t, gv200, gvIda, recs)
	checkContract(t, gt500, gvIda, parseLine(t, gt500, gvIda))
}

// ---- Phase 8: every position-carrying report of both models, scan fallback, GT500 battery cache ----

const (
	// GV200 doc (V5.01 §3.3.1 / §3.3.5 examples, extraction spaces removed)
	gvStt16 = "+RESP:GTSTT,040100,135790246811220,,16,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20090214093254,11F0$"
	gvStr   = "+RESP:GTSTR,060100,135790246811220,,,,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	gvLsp   = "+RESP:GTLSP,060100,135790246811220,,,,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	gvDos   = "+RESP:GTDOS,04040E,862170014697104,,2,1,0,0.1,0,61.3,117.201362,31.832962,20130812114537,0460,0000,5663,5A02,00,20130812114538,0047$"
	gvFla   = "+RESP:GTFLA,040408,135790246811220,,2,92,70,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20090214093254,11F0$"
	gvTmp   = "+RESP:GTTMP,04040B,862170013467608,NMX_Beta,,0,31,1,0,0.4,0,2.2,121.390957,31.164567,20130115083120,0460,0000,1877,0873,00,0.0,00000:00:00,2791,2639,2691,09,09,,,,28967B41040000F1,,25,20130115163122,01AA$"
	gvAis   = "+RESP:GTAIS,040100,135790246811220,,13500,00,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	gvMai   = "+RESP:GTMAI,040100,135790246811220,,1980,11,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	gvLbc   = "+RESP:GTLBC,040100,135790246811220,,+8613800000000,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20090214093254,11F0$"
	gvGin   = "+RESP:GTGIN,060100,135790246811220,,,,100,0,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	gvGot   = "+RESP:GTGOT,060100,135790246811220,,,,100,0,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,2000.0,20090214093254,11F0$"
	// GT500 doc (V0.13 §3.3.1 / §3.3.5)
	gtLbc = "+RESP:GTLBC,072002,135790246811220,,+8613812341234,1,4.3,72,90.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,000ce7000000,-60,1000,80,20090214093254,11F0$"
	gtGcr = "+RESP:GTGCR,072002,135790246811220,,3,50,180,1,4.3,72,90.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,000ce7000000,-60,1000,80,20090214093254,11F0$"
	gtFlc = "+RESP:GTFLC,070009,135790246811220,,,,1,10.3,72,90.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,,0460,0000,1848,1345,54,,0460,0000,1811,0141,53,,0460,0000,1841,0171,43,,0460,0000,1845,0170,33,,0460,0000,1745,0230,33,,0460,0000,1878,0152,16,00,000ce7000000,-56,000ce7000001,-56,000ce7000002,-59,000ce7000003,-68,000ce7000004,-69,000ce7000005,-69,000ce7000006,-70,000ce7000007,-71,000ce7000008,-72,000ce7000009,-73,1000,80,20090214093254,11F0$"
	gtSwg = "+RESP:GTSWG,070002,135790246811220,,1,0,70.0,,,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,000ce7000000,-60,20100214093254,11F0$"
	gtMon = "+RESP:GTMON,070002,135790246811220,,+8613812341234,1,15,7,0,70.0,,,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,000ce7000000,-60,20100214093254,11F0$"
	gtFpo = "+RESP:GTFPO,070002,135790246811220,,1,0,4.3,,,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,000ce7000000,-60,20100214093254,11F0$"
	gtBtc = "+RESP:GTBTC,070002,135790246811220,,0,4.5,,,121.390982,31.164494,20101216085808,0460,0000,1878,0873,00,000ce7000000,-60,20101216085814,0032$"
	gtStc = "+RESP:GTSTC,070002,135790246811220,,,0,4.5,,,121.390982,31.164494,20101216085808,0460,0000,1878,0873,00,000ce7000000,-60,20101216085814,0033$"
	// traccar corpus: other @Track models with the same report names but shorter tails — robustness only
	tSwg    = "+RESP:GTSWG,110100,358688000000158,,1,0,2.1,0,27.1,121.390717,31.164424,20110901073917,0460,0000,1878,0873,,20110901154653,0015$"
	tGcr    = "+RESP:GTGCR,020102,135790246811220,,3,50,180,2,0.4,296,-5.4,121.391055,31.164473,20100714104934,0460,0000,1878,0873,00,,20100714104934,000C$"
	tBplBuf = "+BUFF:GTBPL,1A0800,860599000773978,GL300,3.55,0,0.0,0,257.1,60.565437,56.818277,20161006070553,,,,,204.7,20161006071028,0C75$"
)

var phase8Fixtures = []string{gvStt16, gvStr, gvLsp, gvDos, gvFla, gvTmp, gvAis, gvMai, gvLbc, gvGin, gvGot,
	gtLbc, gtGcr, gtFlc, gtSwg, gtMon, gtFpo, gtBtc, gtStc, tSwg, tGcr, tBplBuf}

func TestGV200Phase8Layouts(t *testing.T) {
	cases := []struct {
		name, line string
		wantOdo    int32
		wantLon    float64
		wantTime   time.Time
	}{
		{"GTSTT doc state 16", gvStt16, 0, 121.354335, utc(2009, 2, 14, 1, 32, 54)},
		{"GTSTR", gvStr, 2000, 121.354335, utc(2009, 2, 14, 1, 32, 54)},
		{"GTSTP", withHeader(gvStr, "+RESP:GTSTP"), 2000, 121.354335, utc(2009, 2, 14, 1, 32, 54)},
		{"GTLSP", gvLsp, 2000, 121.354335, utc(2009, 2, 14, 1, 32, 54)},
		{"+BUFF:GTLSP", withHeader(gvLsp, "+BUFF:GTLSP"), 2000, 121.354335, utc(2009, 2, 14, 1, 32, 54)},
		{"GTDOS", gvDos, 0, 117.201362, utc(2013, 8, 12, 11, 45, 37)},
		{"GTFLA", gvFla, 0, 121.354335, utc(2009, 2, 14, 1, 32, 54)},
		{"GTTMP", gvTmp, 0, 121.390957, utc(2013, 1, 15, 8, 31, 20)},
		{"GTAIS", gvAis, 2000, 121.354335, utc(2009, 2, 14, 1, 32, 54)},
		{"GTMAI", gvMai, 2000, 121.354335, utc(2009, 2, 14, 1, 32, 54)},
		{"GTLBC", gvLbc, 0, 121.354335, utc(2009, 2, 14, 1, 32, 54)},
		{"GTGIN via scan", gvGin, 0, 121.354335, utc(2009, 2, 14, 1, 32, 54)},
		{"GTGOT via scan", gvGot, 0, 121.354335, utc(2009, 2, 14, 1, 32, 54)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recs := parseLine(t, gv200, c.line)
			require.Len(t, recs, 1)
			assert.Equal(t, c.wantOdo, recs[0].Odometer)
			assert.InDelta(t, c.wantLon, recs[0].Position.Longitude, 1e-5)
			assert.Equal(t, c.wantTime, recs[0].Timestamp.AsTime())
			assert.Equal(t, strings.TrimPrefix(splitFields(c.line)[0], "+"), recs[0].MessageType)
		})
	}
	assert.InDelta(t, 13.5, parseLine(t, gv200, gvAis)[0].ControlModuleVoltage, 1e-4, "GTAIS <Analog Input VCC> is the supply voltage")
	assert.Equal(t, float32(0), parseLine(t, gv200, gvMai)[0].ControlModuleVoltage, "GTMAI carries a multi-analog input, not the supply")
	assert.InDelta(t, 4.3, *parseLine(t, gv200, gvStr)[0].Position.Speed, 1e-5)
}

func TestGTTMPTemperature(t *testing.T) {
	r := parseLine(t, gv200, gvTmp)[0]
	assert.InDelta(t, 25, r.Temperature, 1e-6, "<Temperature Sensor device DATA> is decimal °C")
	assert.Equal(t, float32(0), r.ControlModuleVoltage, "Analog Input VCC 0 in the doc example")
	assert.Equal(t, int32(0), r.Odometer)
	r = parseLine(t, gv200, setField(setField(gvTmp, 32, "-12"), 5, "12500"))[0]
	assert.InDelta(t, -12, r.Temperature, 1e-6)
	assert.InDelta(t, 12.5, r.ControlModuleVoltage, 1e-4)
	assert.Equal(t, float32(0), parseLine(t, gv200, setField(gvTmp, 32, ""))[0].Temperature)
	assert.Equal(t, int32(12345), parseLine(t, gv200, setField(gvTmp, 20, "12345.6"))[0].Odometer, "Mileage still read at its GTFRI position")
}

func TestPhase8Alarms(t *testing.T) {
	only := func(vs *types.VehicleStatus) *types.VehicleStatus { return &types.VehicleStatus{Ignition: vs.Ignition} }
	cases := []struct {
		name, line string
		set        func(*types.VehicleStatus)
	}{
		{"GTFLA → fuel theft", gvFla, func(v *types.VehicleStatus) { v.FuelTheft = true }},
		{"+BUFF:GTFLA", withHeader(gvFla, "+BUFF:GTFLA"), func(v *types.VehicleStatus) { v.FuelTheft = true }},
		{"GTDOS → outputs triggering", gvDos, func(v *types.VehicleStatus) { v.OutputsTriggering = true }},
		{"GTLSP → excessive parking", gvLsp, func(v *types.VehicleStatus) { v.ExcessiveParking = true }},
		{"GTGIN → entering geofence", gvGin, func(v *types.VehicleStatus) { v.EntringGeofence = true }},
		{"GTGOT → exiting geofence", gvGot, func(v *types.VehicleStatus) { v.ExitingGeofence = true }},
		{"GTSTR no flag", gvStr, nil}, {"GTSTP no flag", withHeader(gvStr, "+RESP:GTSTP"), nil},
		{"GTSTT no flag", gvStt16, nil}, {"GTAIS no flag", gvAis, nil}, {"GTMAI no flag", gvMai, nil},
		{"GTLBC no flag", gvLbc, nil}, {"GTTMP no flag (temperature is a value)", gvTmp, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recs := parseLine(t, gv200, c.line)
			require.Len(t, recs, 1)
			want := only(recs[0].VehicleStatus)
			if c.set != nil {
				c.set(want)
			}
			assert.Equal(t, want, recs[0].VehicleStatus)
		})
	}
	for _, line := range []string{gtLbc, gtGcr, gtFlc, gtSwg, gtMon, gtFpo, gtBtc, gtStc, gtStt, gtBpl} {
		recs := parseLine(t, gt500, line)
		require.Len(t, recs, 1, line)
		assert.Equal(t, only(recs[0].VehicleStatus), recs[0].VehicleStatus, line)
	}
}

func TestGT500Phase8Layouts(t *testing.T) {
	cases := []struct {
		name, line        string
		wantBatt, wantOdo int32
		wantLon           float64
	}{
		{"GTLBC", gtLbc, 80, 1000, 121.354335},
		{"GTGCR", gtGcr, 80, 1000, 121.354335},
		{"GTFLC variable cell/Wi-Fi tail", gtFlc, 80, 1000, 121.354335},
		{"GTSWG", gtSwg, 0, 0, 121.354335},
		{"GTMON", gtMon, 0, 0, 121.354335},
		{"GTFPO", gtFpo, 0, 0, 121.354335},
		{"GTBTC", gtBtc, 0, 0, 121.390982},
		{"GTSTC", gtStc, 0, 0, 121.390982},
		{"GTSTT", gtStt, 0, 0, 121.354335},
		{"GTBPL", gtBpl, 0, 0, 121.354335},
		{"real GTSWG short tail", tSwg, 0, 0, 121.390717},
		{"real GTGCR short tail", tGcr, 0, 0, 121.391055},
		{"real +BUFF:GTBPL GL300", tBplBuf, 0, 0, 60.565437},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			recs := parseLine(t, gt500, c.line)
			require.Len(t, recs, 1)
			assert.Equal(t, c.wantBatt, recs[0].BatteryLevel)
			assert.Equal(t, c.wantOdo, recs[0].Odometer)
			assert.InDelta(t, c.wantLon, recs[0].Position.Longitude, 1e-5)
			assert.Equal(t, strings.TrimPrefix(splitFields(c.line)[0], "+"), recs[0].MessageType)
		})
	}
	batt := func(recs []*types.DeviceStatus) (out []int32) {
		for _, r := range recs {
			out = append(out, r.BatteryLevel)
		}
		return out
	}
	t.Run("battery is remembered for reports without one", func(t *testing.T) {
		recs, _ := mustStream(t, gt500, f1+gtStt+gtBpl+setField(f1, 21, "42")+gtSwg+gtMon)
		assert.Equal(t, []int32{80, 80, 80, 42, 42, 42}, batt(recs))
	})
	t.Run("GTINF battery percentage feeds the cache", func(t *testing.T) {
		recs, _ := mustStream(t, gt500, gtInf+gtStt)
		assert.Equal(t, []int32{80}, batt(recs), "doc example: 26 fields, percentage 7th from the end")
		tbl := fieldsOf(gtInf)
		tbl = append(tbl[:17], tbl[18:]...) // the PDF's table has one reserved field fewer than its example
		recs, _ = mustStream(t, gt500, join(tbl)+gtStt)
		assert.Equal(t, []int32{80}, batt(recs), "table shape: 25 fields")
		for _, bad := range []string{"abc", "", "101", "-1"} {
			recs, _ = mustStream(t, gt500, f1+setField(gtInf, 19, bad)+gtStt)
			assert.Equal(t, []int32{80, 80}, batt(recs), "invalid %q keeps the last value", bad)
		}
	})
	t.Run("tail length mismatch keeps the cached battery", func(t *testing.T) {
		recs, _ := mustStream(t, gt500, f1+insertAfter(f1, 22, ""))
		assert.Equal(t, []int32{80, 80}, batt(recs))
	})
	t.Run("battery is per connection", func(t *testing.T) {
		mustStream(t, gt500, f1)
		recs, _ := mustStream(t, gt500, gtStt)
		assert.Equal(t, []int32{0}, batt(recs))
	})
}

func TestScanFallback(t *testing.T) {
	t.Run("unknown GV200 header with a GPS block", func(t *testing.T) {
		recs := parseLine(t, gv200, withHeader(gvMpn, "+RESP:GTXYZ"))
		require.Len(t, recs, 1)
		assert.Equal(t, "RESP:GTXYZ", recs[0].MessageType)
		assert.InDelta(t, 31.222073, recs[0].Position.Latitude, 1e-5)
		assert.InDelta(t, 4.3, *recs[0].Position.Speed, 1e-5)
		assert.Equal(t, float32(92), recs[0].Position.Course)
		assert.Equal(t, utc(2009, 2, 14, 1, 32, 54), recs[0].Timestamp.AsTime())
		assert.Equal(t, int32(0), recs[0].Odometer, "scanned reports have no tail parsing")
	})
	t.Run("unknown GT500 header", func(t *testing.T) {
		recs := parseLine(t, gt500, withHeader(gtBtc, "+BUFF:GTXYZ"))
		require.Len(t, recs, 1)
		assert.InDelta(t, 121.390982, recs[0].Position.Longitude, 1e-5)
		assert.Equal(t, "BUFF:GTXYZ", recs[0].MessageType)
	})
	t.Run("append mask honoured on scanned GV200 reports", func(t *testing.T) {
		recs := parseLine(t, gv200, withHeader(tIgnBuf, "+BUFF:GTXYZ"))
		require.Len(t, recs, 1)
		assert.Equal(t, int32(11), recs[0].Position.Satellites)
	})
	t.Run("no GPS block → counted as ignored", func(t *testing.T) {
		p := &Protocol{DeviceType: gv200, Imei: imei}
		for _, line := range []string{
			"+RESP:GTALL,040100,135790246811220,,GEO,0,3,121.354335,31.222073,1000,600,0,,,,SRI,1,3,1,116.238.112.142,7005,20090214093254,11F0$", // geo-fence centre: a radius follows, not a time
			"+RESP:GTGSM,040100,135790246811220,,FRI,0460,0000,18d8,6141,20,,0460,0000,1878,0152,16,,20090214093254,11F0$",
			"+RESP:GTXYZ,040100,135790246811220,,121.354335,31.222073,20090214013254$",                   // no room for HDOP…altitude before the pair
			"+RESP:GTXYZ,040100,135790246811220,,1,2,3,4,121.354335,31.222073,notatime,0460,0000$",       // no 14-char time after the pair
			"+RESP:GTXYZ,040100,135790246811220,,1,2,3,4,121.354335,31.222073,20091399999999,0460,0000$", // 14 chars but not a time
			"+RESP:GTXYZ,040100,135790246811220,,1,2,3,4,121.354335,31.222073$",                          // truncated
			"+RESP:GTXYZ,040100,135790246811220,,1,2,3,4,0.000000,0.000000,20090214013254,0460,0000$",    // no fix
		} {
			assert.Empty(t, p.parseReport(splitFields(strings.TrimSuffix(line, "$")), nil), line)
		}
		assert.Equal(t, map[string]int{"+RESP:GTALL": 1, "+RESP:GTGSM": 1, "+RESP:GTXYZ": 5}, p.ignored)
	})
	t.Run("an explicit Number=0 on a trusted layout is honoured, no scan", func(t *testing.T) {
		recs := parseLine(t, gv200, setField(gv1, 6, "0")) // the device said "no points", even though a block is present
		assert.Empty(t, recs)
	})
}

func TestContractOnPhase8Fixtures(t *testing.T) {
	for _, dt := range []types.DeviceType{gt500, gv200} {
		for _, line := range phase8Fixtures {
			checkContract(t, dt, line, parseLine(t, dt, line))
		}
	}
}

// ---- Multi-model registry + GL300 family ----
// "doc" lines are verbatim from "GL300 @Tracker Air Interface Protocol V6.00" (TRACGL300AN008);
// "traccar" lines are real GL300/GL300VC/GL300W traffic from Traccar's corpus (Apache-2.0,
// © Anton Tananaev); "gw" lines are the live tstGW gateway captured on production (2026-09-01).

const (
	gl300 = types.DeviceType_QUECLINK_GL300

	gl3Fri   = "+RESP:GTFRI,1A0600,135790246811220,,0,0,1,1,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,,20090214093254,11F0$" // doc §3.3.1
	gl3Igl   = "+RESP:GTIGL,1A0600,867844000125073,,,00,1,5,,,,117.201362,31.832724,20120821032037,,,,,,,,000C$"                                          // doc §3.3.1
	gl3Btc   = "+RESP:GTBTC,1A0600,135790246811220,,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20100214093254,11F0$"        // doc §3.3.4
	gl3Ign   = "+RESP:GTIGN,1A0600,135790246811220,,1200,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20100214093254,11F0$"   // doc §3.3.4 (tail completed like the other event examples)
	gl3Igf   = "+RESP:GTIGF,1A0600,135790246811220,,1200,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20100214093254,11F0$"
	gl3Fri4  = "+RESP:GTFRI,1A0900,860599000306845,G3-313,0,0,4,1,2.1,0,426.7,8.611466,47.681639,20181214134603,0228,0001,077F,4812,25.2,1,5.7,34,437.3,8.611600,47.681846,20181214134619,0228,0001,077F,4812,25.2,1,4.4,62,438.2,8.611893,47.681983,20181214134633,0228,0001,077F,4812,25.2,1,4.8,78,436.6,8.612236,47.682040,20181214134648,0228,0001,077F,4812,25.2,83,20181214134702,0654$" // traccar
	gl3Fri99 = "+RESP:GTFRI,1A0401,860599000508846,,0,0,1,1,134.8,154,278.7,-76.671089,39.778885,20150623154301,0310,0260,043F,7761,,99,20150623154314,0F24$"                                                                                                                                                                                                                                   // traccar
	gl3FriW  = "+RESP:GTFRI,2C0402,867162020000816,,0,0,1,2,0.3,337,245.7,-82.373387,34.634011,20170215003054,,,,,,63,20170215003241,3EAB$"                                                                                                                                                                                                                                                     // traccar GL300W
	gl3Stt   = "+RESP:GTSTT,1A0401,860599000508846,,41,0,0.0,84,107.5,-76.657998,39.497203,20150623160622,0310,0260,B435,3B81,,20150623160622,0F54$"                                                                                                                                                                                                                                            // traccar
	gl3SttVC = "+RESP:GTSTT,280100,A1000043D20139,,42,0,0.1,321,228.6,-76.660884,39.832552,20150615120628,0310,0484,00600019,0A52,,20150615085741,0320$"                                                                                                                                                                                                                                        // traccar GL300VC
	gl3RtlVC = "+RESP:GTRTL,280100,A1000043D20139,,0,0,1,1,0.1,321,239.1,-76.661047,39.832501,20150615114455,0310,0484,00600019,0A52,,87,20150615074456,031E$"                                                                                                                                                                                                                                  // traccar GL300VC
	gl3Bpl   = "+BUFF:GTBPL,1A0800,860599000773978,GL300,3.55,0,0.0,0,257.1,60.565437,56.818277,20161006070553,,,,,204.7,20161006071028,0C75$"                                                                                                                                                                                                                                                  // traccar
	gl3Tem   = "+RESP:GTTEM,1A0102,860599000000448,,3,33,0,5.8,0,33.4,117.201191,31.832502,20130109061410,0460,0000,5678,2079,,20130109061517,0091$"                                                                                                                                                                                                                                            // traccar
	gl3Tsw   = "+RESP:GTTSW,1A0100,135790246811220,,1,0,0,4.3,92,70.0,121.354335,31.222073,20090214013254,0460,0000,18d8,6141,00,20100214093254,11F0$"                                                                                                                                                                                                                                          // traccar
	gl3Inf   = "+RESP:GTINF,1A0800,860599000773978,GL300,41,89701016426133851978,17,0,1,26.6,,3.90,1,1,0,0,0,20161003184043,69,1,44,,,20161004040811,022C$"                                                                                                                                                                                                                                     // traccar: battery 69 at len-7

	gwFri = "+RESP:GTFRI,280518,863922034601352, tstGW v1.0.43.11 S25 B100% C2|G V11 H1 A141m D0 0kph 11634m|,,,,1.1,0,0,,78.021396,27.202109,20260901080800,,,,,,,11634,100,20260901180828,0004$"
	gwStc = "+RESP:GTSTC,280518,863922038155132, tstGW v1.0.43.11 S26 B100% C2|G V10 H0.8 A15m D0 1kph 2561810m|,,0.8,0,0,,151.501029,-33.347904,20260901090510,,,,,,,,20260901190955,000B$"
	gwBtc = "+RESP:GTBTC,280518,863922038155132, tstGW v1.0.43.11 S26 B100% C2|G V10 H0.8 A15m D0 1kph 2561810m|,0.8,0,0,,151.501029,-33.347904,20260901090515,,,,,,,,20260901190955,000C$"
	gwDis = "+RESP:GTDIS,280518,863922034601352, tstGW v1.0.43.11 S25 B100% C2|G V13 H1.1 A109m D70 2kph 11634m|,,71,,1.1,0,70,,78.021260,27.202053,20260901183359,,,,,,11634,20260902043610,0021$"

	// "cli" lines are the client's 2021 sample ("Tracker data.xlsx"), older firmware of the same units.
	cliFri = "+RESP:GTFRI,280518,863922032566912,G S21 V10 B100 C0 H14.6 v1.0.29.3,,,,14.6,21,250,,151.196116,-33.793433,20210930033650,,,,,,,3711484,100,20210930133655,0023$"
	cliSos = "+RESP:GTSOS,280518,863922032566912,G S22 V8 B100 C0 H20.2 v1.0.29.3,,,,19.9,1,0,,151.198821,-33.791755,20210930033245,,,,,,,3711160,,20210930133257,001A$"
	cliDis = "+BUFF:GTDIS,280518,863921033970289,OB S17 V0 B99 C0 H0 v1.0.29.3,,91,,0,0,0,,151.190258,-33.785535,20210930030952,,,,,,0,20210930130957,001B$"
	cliStc = "+RESP:GTSTC,280518,863921033970289,B S27 V0 B99 C0 H1 v1.0.29.3,,1,0,0,,151.190258,-33.785535,20210930030515,,,,,,,,20210930130523,0019$"
	cliBtc = "+RESP:GTBTC,280518,863922032566235,B S24 V0 B40 C1 H1 v1.0.29.3,1,0,0,,151.190367,-33.785525,20210930041319,,,,,,,,20210930141330,0003$"
)

var gl300Fixtures = []string{gl3Fri, gl3Igl, gl3Btc, gl3Ign, gl3Igf, gl3Fri4, gl3Fri99, gl3FriW,
	gl3Stt, gl3SttVC, gl3RtlVC, gl3Bpl, gl3Tem, gl3Tsw, gl3Inf, gwFri, gwStc, gwBtc, gwDis, cliFri, cliSos, cliDis, cliStc, cliBtc}

// parseWith parses one line on a connection whose device announced the given version prefix.
func parseWith(t *testing.T, dt types.DeviceType, prefix, line string) []*types.DeviceStatus {
	t.Helper()
	raw := strings.TrimSuffix(line, "$")
	return (&Protocol{DeviceType: dt, Imei: imei, versionPrefix: prefix}).parseReport(splitFields(raw), []byte(raw))
}

func TestGL300Position(t *testing.T) { // doc §3.3.1 example, verbatim
	recs := parseLine(t, gl300, gl3Fri)
	require.Len(t, recs, 1)
	r := recs[0]
	assert.InDelta(t, 31.222073, r.Position.Latitude, 1e-5)
	assert.InDelta(t, 121.354335, r.Position.Longitude, 1e-5)
	assert.Equal(t, utc(2009, time.February, 14, 1, 32, 54), r.Timestamp.AsTime())
	assert.InDelta(t, 4.3, *r.Position.Speed, 1e-5)
	assert.InDelta(t, 92, r.Position.Course, 1e-5)
	assert.InDelta(t, 70.0, r.Position.Altitude, 1e-5)
	assert.Equal(t, int32(0), r.Odometer)     // per-block odo "00"
	assert.Equal(t, int32(0), r.BatteryLevel) // battery field empty in the doc example
	assert.Equal(t, "RESP:GTFRI", r.MessageType)
}

func TestGL300MultiPoint(t *testing.T) { // every block carries its own odo; battery once, at the end
	recs := parseLine(t, gl300, gl3Fri4)
	require.Len(t, recs, 4)
	assert.Equal(t, utc(2018, time.December, 14, 13, 46, 3), recs[0].Timestamp.AsTime())
	assert.Equal(t, utc(2018, time.December, 14, 13, 46, 48), recs[3].Timestamp.AsTime())
	for _, r := range recs {
		assert.Equal(t, int32(83), r.BatteryLevel)
		assert.Equal(t, int32(25), r.Odometer) // last block's odo 25.2 km, floored
		assert.InDelta(t, 47.68, r.Position.Latitude, 0.01)
	}
}

func TestGL300Headers(t *testing.T) {
	for _, tc := range []struct {
		line     string
		lat, lon float32
		battery  int32
	}{
		{gl3Fri99, 39.778885, -76.671089, 99},
		{gl3FriW, 34.634011, -82.373387, 63}, // GL300W: empty cell info and odo
		{gl3RtlVC, 39.832501, -76.661047, 87},
		{gl3Stt, 39.497203, -76.657998, 0}, // event report: no battery in the tail
		{gl3Bpl, 56.818277, 60.565437, 0},  // +BUFF battery-low with position
		{gl3Btc, 31.222073, 121.354335, 0}, // charging event
		{gl3Tsw, 31.222073, 121.354335, 0}, // tamper switch
		{gl3Igl, 31.832724, 117.201362, 0}, // ignition-on location, mostly-empty fields
	} {
		recs := parseLine(t, gl300, tc.line)
		require.Len(t, recs, 1, tc.line)
		assert.InDelta(t, tc.lat, recs[0].Position.Latitude, 1e-5, tc.line)
		assert.InDelta(t, tc.lon, recs[0].Position.Longitude, 1e-5, tc.line)
		assert.Equal(t, tc.battery, recs[0].BatteryLevel, tc.line)
	}
}

func TestGL300Temperature(t *testing.T) {
	recs := parseLine(t, gl300, gl3Tem)
	require.Len(t, recs, 1)
	assert.InDelta(t, 33, recs[0].Temperature, 1e-5) // <Temperature> at #5, °C
}

func TestGL300Ignition(t *testing.T) {
	t.Run("GTSTT motion states are the ignition proxy", func(t *testing.T) {
		p := &Protocol{DeviceType: gl300, Imei: imei}
		recs := p.parseReport(fieldsOf(gl3SttVC), nil) // state 42 = moving
		require.Len(t, recs, 1)
		assert.True(t, *recs[0].VehicleStatus.Ignition)
		recs = p.parseReport(fieldsOf(gl3Stt), nil) // state 41 = motionless
		require.Len(t, recs, 1)
		assert.False(t, *recs[0].VehicleStatus.Ignition)
	})
	t.Run("wired GTIGN/GTIGF/GTIGL win (GL300VC)", func(t *testing.T) {
		p := &Protocol{DeviceType: gl300, Imei: imei}
		recs := p.parseReport(fieldsOf(gl3Ign), nil)
		require.Len(t, recs, 1)
		assert.True(t, *recs[0].VehicleStatus.Ignition)
		recs = p.parseReport(fieldsOf(gl3Igf), nil)
		require.Len(t, recs, 1)
		assert.False(t, *recs[0].VehicleStatus.Ignition)
		recs = p.parseReport(fieldsOf(gl3Igl), nil) // report type 0 = ignition on
		require.Len(t, recs, 1)
		assert.True(t, *recs[0].VehicleStatus.Ignition)
	})
	t.Run("a GV200 keeps its ignition line for GTSTT 41/42", func(t *testing.T) {
		p := &Protocol{DeviceType: gv200, Imei: imei}
		p.parseReport(fieldsOf(gvStt520), nil) // state 42: motion sensor, no ignition signal
		assert.Nil(t, p.lastIgnition)          // speed rule stays
	})
}

func TestGL300Info(t *testing.T) { // GTINF: CSQ 17 → level 3, battery 69 at len-7, state 41
	p := &Protocol{DeviceType: gl300, Imei: imei}
	assert.Empty(t, p.parseReport(fieldsOf(gl3Inf), nil))
	assert.Equal(t, int32(3), p.gsmLevel())
	assert.Equal(t, int32(69), p.battery)
	require.NotNil(t, p.lastIgnition)
	assert.False(t, *p.lastIgnition)
	recs := p.parseReport(fieldsOf(gl3Fri), nil) // battery-less position reuses the cached 69
	require.Len(t, recs, 1)
	assert.Equal(t, int32(69), recs[0].BatteryLevel)
}

func TestPrefixWinsOverRegistration(t *testing.T) {
	// Registered as GV200, announces GL300 (prefix 1A): the wire's layout is used — battery 99
	// sits where the GL300 tail says, which a GV200 parse would never read.
	recs := parseWith(t, gv200, "1A", gl3Fri99)
	require.Len(t, recs, 1)
	assert.Equal(t, int32(99), recs[0].BatteryLevel)
	assert.Equal(t, gv200, recs[0].DeviceType) // records keep the registered type
	// And the reverse: registered as GL300, announces GV200 (prefix 04).
	recs = parseWith(t, gl300, "04", gv1)
	require.Len(t, recs, 1)
	assert.Equal(t, int32(2000), recs[0].Odometer) // GV200 tail: Mileage right after the block
}

func TestGatewayLines(t *testing.T) { // live tstGW capture: prefix 28 → gwSpec (GT500 layouts + GV300W GTDIS)
	r := bufio.NewReader(strings.NewReader(gwFri + gwStc + gwBtc + gwDis))
	p := &Protocol{}
	_, _, err := p.Login(r)
	require.NoError(t, err)
	p.SetDeviceType(gv200) // mis-registered, like production
	st := newFakeStore()
	require.ErrorIs(t, p.ConsumeStream(r, io.Discard, st), io.EOF)
	recs := st.drain()
	require.Len(t, recs, 4)
	// GTFRI: blank Number = one block at the GT500 position; tail Odo (metres → 11 km), Batt%;
	// satellites and GSM level from the device-name status ("S25 … V11").
	assert.InDelta(t, 27.202109, recs[0].Position.Latitude, 1e-5)
	assert.InDelta(t, 78.021396, recs[0].Position.Longitude, 1e-5)
	assert.Equal(t, utc(2026, time.September, 1, 8, 8, 0), recs[0].Timestamp.AsTime())
	assert.Equal(t, int32(100), recs[0].BatteryLevel)
	assert.Equal(t, int32(11), recs[0].Odometer)
	assert.Equal(t, int32(11), recs[0].Position.Satellites)
	assert.Equal(t, int32(5), recs[0].GsmNetwork)
	// GTSTC / GTBTC: GT500 event shape (Reserved,MAC,RSSI,SendTime,Count tail); no battery field,
	// the name's "B100%" fills it.
	assert.InDelta(t, -33.347904, recs[1].Position.Latitude, 1e-5)
	assert.Equal(t, utc(2026, time.September, 1, 9, 5, 10), recs[1].Timestamp.AsTime())
	assert.Equal(t, int32(100), recs[1].BatteryLevel)
	assert.Equal(t, int32(10), recs[1].Position.Satellites)
	assert.InDelta(t, -33.347904, recs[2].Position.Latitude, 1e-5)
	assert.Equal(t, utc(2026, time.September, 1, 9, 5, 15), recs[2].Timestamp.AsTime())
	assert.Equal(t, int32(100), recs[2].BatteryLevel)
	// GTDIS: GV300W shape; the tail Mileage (11634) must not be mistaken for a battery, and the
	// odometer is left to GTFRI. Code 71 = self-test report (client's event sheet).
	assert.InDelta(t, 27.202053, recs[3].Position.Latitude, 1e-5)
	assert.Equal(t, float32(70), recs[3].Position.Course)
	assert.True(t, recs[3].VehicleStatus.SelfTest)
	assert.False(t, recs[3].VehicleStatus.InputsTriggering)
	assert.Equal(t, int32(100), recs[3].BatteryLevel)
	assert.Equal(t, int32(0), recs[3].Odometer)
	assert.Equal(t, int32(13), recs[3].Position.Satellites)
}

func TestPrefixOf(t *testing.T) {
	assert.Equal(t, "1A", prefixOf("1A0600"))
	assert.Equal(t, "28", prefixOf("280518"))
	assert.Equal(t, "802004", prefixOf("8020040200")) // 6-char model code
	assert.Equal(t, "80", prefixOf("80FF"))           // 6-char lookup misses → first two
	assert.Equal(t, "", prefixOf("X"))
}

func TestModelName(t *testing.T) {
	assert.Equal(t, "QUECLINK_GL300", modelName("28"))
	assert.Equal(t, "GT501 (no field layouts — positions are located by field scan)", modelName("42"))
	assert.Equal(t, "unknown Queclink model", modelName("ZZ"))
}

func TestLayoutTrusted(t *testing.T) {
	for prefix, want := range map[string]bool{
		"": true, "04": true, "35": true, "07": true, "1A": true, "28": true, "2C": true,
		"42": false, "1F": false, "ZZ": false,
	} {
		assert.Equal(t, want, (&Protocol{versionPrefix: prefix}).layoutTrusted(), prefix)
	}
}

func TestUnknownModelDoesNotTrustTheNumberField(t *testing.T) {
	// A GT501 (prefix 42, no layouts held) has other data where our tables expect Number; a "0"
	// there must not suppress the scan. On a trusted layout an explicit 0 still means "no points".
	line := setField(gl3Fri99, 6, "0")
	recs := parseWith(t, gl300, "42", line)
	require.Len(t, recs, 1)                          // rescued by scan
	assert.Equal(t, int32(99), recs[0].BatteryLevel) // the end-anchored battery survives the scan
	assert.Equal(t, int32(0), recs[0].Odometer)      // mid-report telemetry does not
	assert.Empty(t, parseWith(t, gl300, "1A", line)) // trusted layout: 0 points honoured
}

func TestContractOnGL300Fixtures(t *testing.T) {
	for _, line := range gl300Fixtures {
		p := &Protocol{DeviceType: gl300, Imei: imei}
		raw := strings.TrimSuffix(line, "$")
		recs := p.parseReport(splitFields(raw), []byte(raw))
		for _, r := range recs {
			require.NotNil(t, r.Position, line)
			require.NotNil(t, r.VehicleStatus.Ignition, line)
			assert.Equal(t, gl300, r.DeviceType, line)
			assert.Equal(t, time.UTC, r.Timestamp.AsTime().Location(), line)
			assert.NotNil(t, r.GetQueclinkPacket(), line)
		}
	}
}

func TestGatewayNameStatus(t *testing.T) {
	// Both firmware generations pack the status into <Device name>; only S/V/B are mapped.
	for _, tc := range []struct {
		name            string
		gsm, batt, sats int32
	}{
		{" tstGW v1.0.43.11 S25 B100% C2|G V11 H1 A141m D0 0kph 11634m|", 5, 100, 11},
		{" tstGW v1.0.43.11 S26 B100% C2|OG |", 5, 100, -1}, // buffered with an old fix: no satellite count
		{"B S30 V0 B100 C1 H1 v1.0.29.3", 5, 100, 0},        // 2021 firmware, Bluetooth fix
		{"OB S17 V0 B99 C0 H0 v1.0.29.3", 3, 99, 0},         // the method letters never match as B<n>
		{"G S15 V11 B12 C0 H10.4 v1.0.29.2", 3, 12, 11},
	} {
		p := &Protocol{}
		assert.Equal(t, tc.sats, p.readNameStatus(tc.name), tc.name)
		assert.Equal(t, tc.gsm, p.gsmLevel(), tc.name)
		assert.Equal(t, tc.batt, p.battery, tc.name)
	}
	p := &Protocol{}
	assert.Equal(t, int32(-1), p.readNameStatus("GL300")) // a plain device name states nothing
	assert.Equal(t, int32(-1), p.gsmLevel())
	assert.Equal(t, int32(0), p.battery)
	// Only the gateway spec reads the name: a GT500 whose name happens to look like it is untouched.
	recs := parseLine(t, gt500, setField(f1, 3, "S30 V9 B55"))
	require.Len(t, recs, 1)
	assert.Equal(t, int32(80), recs[0].BatteryLevel)
	assert.Equal(t, int32(0), recs[0].Position.Satellites)
	assert.Equal(t, int32(-1), recs[0].GsmNetwork)
}

func TestGatewayOdometerAndBattery(t *testing.T) {
	// GTFRI: GT500 tail, odometer in metres → km, battery from the tail; V5 → satellites.
	recs := parseWith(t, gl300, "28", cliFri)
	require.Len(t, recs, 1)
	assert.Equal(t, int32(3711), recs[0].Odometer) // 3711484 m
	assert.Equal(t, int32(100), recs[0].BatteryLevel)
	assert.Equal(t, int32(10), recs[0].Position.Satellites)
	assert.InDelta(t, 21, *recs[0].Position.Speed, 1e-5) // #7 is HDOP (14.6), #8 speed, #9 course
	assert.Equal(t, float32(250), recs[0].Position.Course)
	assert.True(t, *recs[0].VehicleStatus.Ignition) // moving: speed rule, no motion-sensor report yet
	// GTSOS: same shape with the tail battery empty → the name's B100 stands in.
	recs = parseWith(t, gl300, "28", cliSos)
	require.Len(t, recs, 1)
	assert.True(t, recs[0].VehicleStatus.SosButtonPressed)
	assert.Equal(t, int32(100), recs[0].BatteryLevel)
	assert.Equal(t, int32(3711), recs[0].Odometer)
	// GTDIS with Mileage 0: previously read as battery 0 by the end-anchored scan path. Code 91 = check-in.
	recs = parseWith(t, gl300, "28", cliDis)
	require.Len(t, recs, 1)
	assert.True(t, recs[0].VehicleStatus.CheckIn)
	assert.Equal(t, int32(99), recs[0].BatteryLevel)
	assert.Equal(t, int32(0), recs[0].Odometer)
	assert.InDelta(t, -33.785535, recs[0].Position.Latitude, 1e-5)
	assert.Equal(t, utc(2021, time.September, 30, 3, 9, 52), recs[0].Timestamp.AsTime())
	// GTSTC/GTBTC: battery 99 / 40 from the name (no battery field in these reports).
	assert.Equal(t, int32(99), parseWith(t, gl300, "28", cliStc)[0].BatteryLevel)
	assert.Equal(t, int32(40), parseWith(t, gl300, "28", cliBtc)[0].BatteryLevel)
	// The spec-conformant GL300 layouts are untouched: kilometres stay kilometres.
	recs = parseLine(t, gl300, gl3Fri4)
	require.NotEmpty(t, recs)
	assert.Equal(t, int32(25), recs[0].Odometer)
	// And a real GL300VC line on prefix 28 still yields its position (block at the same index).
	recs = parseWith(t, gl300, "28", gl3RtlVC)
	require.Len(t, recs, 1)
	assert.InDelta(t, 39.832501, recs[0].Position.Latitude, 1e-5)
}

// TestGatewayEvents: every row of the client's event sheet that the EC01 sends, one flag each.
func TestGatewayEvents(t *testing.T) {
	flags := func(vs *types.VehicleStatus) []string {
		var out []string
		for _, f := range []struct {
			name string
			on   bool
		}{
			{"monitoring_off", vs.MonitoringOff}, {"monitoring_on", vs.MonitoringOn}, {"tilt_alert", vs.TiltAlert},
			{"fall_detected", vs.FallDetected}, {"no_motion_alert", vs.NoMotionAlert}, {"self_test", vs.SelfTest},
			{"welfare_alarm", vs.WelfareAlarm}, {"check_in_reminder", vs.CheckInReminder}, {"check_out", vs.CheckOut},
			{"check_in", vs.CheckIn}, {"leave_home", vs.LeaveHome}, {"arrive_home", vs.ArriveHome},
			{"inputs_triggering", vs.InputsTriggering}, {"battery_low", vs.BatteryLow}, {"charging_started", vs.ChargingStarted},
			{"charging_stopped", vs.ChargingStopped}, {"motion_alert", vs.MotionAlert}, {"sos_button_pressed", vs.SosButtonPressed},
			{"over_speeding", vs.OverSpeeding}, {"entering_geofence", vs.EntringGeofence}, {"exiting_geofence", vs.ExitingGeofence},
		} {
			if f.on {
				out = append(out, f.name)
			}
		}
		return out
	}
	// GTDIS <ReportID/Type> is hex: input nibble then state nibble.
	for code, want := range map[string]string{
		"40": "monitoring_off", "41": "monitoring_on", "50": "tilt_alert", "51": "fall_detected",
		"70": "no_motion_alert", "71": "self_test", "80": "welfare_alarm", "81": "check_in_reminder",
		"90": "check_out", "91": "check_in", "A0": "leave_home", "A1": "arrive_home",
		// reference-only / undocumented codes stay generic
		"10": "inputs_triggering", "21": "inputs_triggering", "61": "inputs_triggering", "FF": "inputs_triggering", "": "inputs_triggering",
	} {
		recs := parseWith(t, gl300, "28", setField(cliDis, 5, code))
		require.Len(t, recs, 1, code)
		assert.Equal(t, []string{want}, flags(recs[0].VehicleStatus), "code %q", code)
	}
	// Other event reports in the GT500 shape (block at #5 after one leading field, or #4).
	assert.Equal(t, []string{"charging_started"}, flags(parseWith(t, gl300, "28", cliBtc)[0].VehicleStatus))
	assert.Equal(t, []string{"charging_stopped"}, flags(parseWith(t, gl300, "28", cliStc)[0].VehicleStatus))
	assert.Equal(t, []string{"battery_low"}, flags(parseWith(t, gl300, "28", setField(withHeader(cliStc, "+RESP:GTBPL"), 4, "3.53"))[0].VehicleStatus))
	// GTSTT: <State> at #4; x2 = moving → motion alert, x1 = at rest → nothing.
	assert.Equal(t, []string{"motion_alert"}, flags(parseWith(t, gl300, "28", setField(withHeader(cliStc, "+RESP:GTSTT"), 4, "42"))[0].VehicleStatus))
	assert.Empty(t, flags(parseWith(t, gl300, "28", setField(withHeader(cliStc, "+RESP:GTSTT"), 4, "41"))[0].VehicleStatus))
	// The shared mapping still applies to the rest.
	assert.Equal(t, []string{"sos_button_pressed"}, flags(parseWith(t, gl300, "28", cliSos)[0].VehicleStatus))
	assert.Equal(t, []string{"over_speeding"}, flags(parseWith(t, gl300, "28", withHeader(cliFri, "+RESP:GTSPD"))[0].VehicleStatus))
	assert.Equal(t, []string{"exiting_geofence"}, flags(parseWith(t, gl300, "28", withHeader(cliFri, "+RESP:GTGEO"))[0].VehicleStatus))
	assert.Equal(t, []string{"entering_geofence"}, flags(parseWith(t, gl300, "28", setField(withHeader(cliFri, "+RESP:GTGEO"), 5, "1"))[0].VehicleStatus))
	// Non-gateway models are untouched: a GL300 GTDIS keeps the generic flag.
	recs := parseLine(t, gl300, withHeader(gl3Fri99, "+RESP:GTDIS"))
	require.Len(t, recs, 1)
	assert.Equal(t, []string{"inputs_triggering"}, flags(recs[0].VehicleStatus))
}

// TestClientSampleAndProductionLines replays the client's 2021 sample and every production line
// of 2026-09-01/02 (testdata/) and checks each record against the client's own field map
// ("Tracker data.xlsx"): exactly one record per line; position, speed and course from the
// fields the map names; battery, satellites and GSM level from the device-name status;
// odometer = tail metres / 1000 on GTFRI/GTSOS and unset elsewhere; GTDIS → inputs_triggering,
// GTSOS → SOS. GTFRI/GTSOS/GTDIS have the block at #7, GTSTC at #5, GTBTC at #4.
func TestClientSampleAndProductionLines(t *testing.T) {
	statusRe := regexp.MustCompile(`\b([SVB])(\d+)\b`)
	for file, wantLines := range map[string]int{"testdata/client-tracker-data-2021.txt": 80, "testdata/bobcat-2026-09-01.txt": 47} {
		data, err := os.ReadFile(file)
		require.NoError(t, err)
		n := 0
		for _, line := range strings.Split(string(data), "\n") {
			if line = strings.TrimSpace(line); line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			n++
			f := fieldsOf(line)
			report := f[0][6:]
			lon := map[string]int{"GTFRI": 11, "GTSOS": 11, "GTDIS": 11, "GTSTC": 9, "GTBTC": 8}[report]
			require.NotZero(t, lon, "unexpected header in %s", line)
			recs := parseWith(t, gl300, "28", line)
			require.Len(t, recs, 1, line)
			r := recs[0]
			assert.InDelta(t, atof(f[lon]), r.Position.Longitude, 1e-5, line)
			assert.InDelta(t, atof(f[lon+1]), r.Position.Latitude, 1e-5, line)
			ts, err := time.Parse("20060102150405", f[lon+2])
			require.NoError(t, err, line)
			assert.Equal(t, ts, r.Timestamp.AsTime(), line)
			assert.InDelta(t, atof(f[lon-3]), *r.Position.Speed, 1e-5, line)
			assert.InDelta(t, atof(f[lon-2]), r.Position.Course, 1e-5, line)
			sats, batt, gsm := int32(0), int32(0), int32(-1)
			for _, m := range statusRe.FindAllStringSubmatch(f[3], -1) {
				switch m[1] {
				case "S":
					gsm, _ = csqLevel(m[2])
				case "V":
					sats = int32(atoi(m[2]))
				case "B":
					batt = int32(atoi(m[2]))
				}
			}
			assert.Equal(t, batt, r.BatteryLevel, line)
			assert.Equal(t, sats, r.Position.Satellites, line)
			assert.Equal(t, gsm, r.GsmNetwork, line)
			odo := int32(0)
			if report == "GTFRI" || report == "GTSOS" {
				odo = int32(atoi(f[len(f)-4]) / 1000)
			}
			assert.Equal(t, odo, r.Odometer, line)
			// events: the client's sheet maps each report / GTDIS code to one flag; the sample's codes
			// are 90 (check-out), 91 (check-in), 80 (welfare alarm) and, in production, 71 (self-test).
			want := map[string]bool{"GTSOS": true}[report]
			assert.Equal(t, want, r.VehicleStatus.SosButtonPressed, line)
			assert.Equal(t, report == "GTBTC", r.VehicleStatus.ChargingStarted, line)
			assert.Equal(t, report == "GTSTC", r.VehicleStatus.ChargingStopped, line)
			if report == "GTDIS" {
				flag := map[string]bool{"90": r.VehicleStatus.CheckOut, "91": r.VehicleStatus.CheckIn, "80": r.VehicleStatus.WelfareAlarm, "71": r.VehicleStatus.SelfTest}[f[5]]
				assert.True(t, flag, "GTDIS code %s not mapped: %s", f[5], line)
				assert.False(t, r.VehicleStatus.InputsTriggering, line)
			} else {
				assert.False(t, r.VehicleStatus.InputsTriggering, line)
			}
			assert.Equal(t, strings.TrimSuffix(line, "$"), string(r.GetQueclinkPacket().GetRawData()), line)
		}
		assert.Equal(t, wantLines, n, file)
	}
}
