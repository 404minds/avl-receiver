package tr06

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/404minds/avl-receiver/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Vectors are of two kinds:
//
//   - "spec" - packets printed in the Concox TR06 protocol document (§5.2.2, §5.3.1.21 and
//     Appendix B). They pin the decoder to the specification, CRC included.
//   - "gs21" - frames captured on the wire from WanWay GS21 IMEI 865948050016579
//     (/root/tr06-capture on the receiver host). They pin the decoder to the real device.
//
// Every frame carries its own serial number and CRC exactly as transmitted, so a
// successful parsePacket also exercises the framing and CRC path.

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	require.NoError(t, err)
	return b
}

func parseFrame(t *testing.T, frame string) *Packet {
	t.Helper()
	p := &TR06Protocol{}
	packet, err := p.parsePacket(bufio.NewReader(bytes.NewReader(mustHex(t, frame))))
	require.NoError(t, err, "frame must parse and pass CRC")
	return packet
}

func parseBody(t *testing.T, body string) *Packet {
	t.Helper()
	p := &TR06Protocol{}
	packet := &Packet{}
	require.NoError(t, p.parsePacketData(bufio.NewReader(bytes.NewReader(mustHex(t, body))), packet))
	return packet
}

func TestLoginPacket(t *testing.T) {
	cases := []struct {
		name     string
		frame    string
		wantIMEI string
		wantAck  string
	}{
		// TR06 Appendix B: login packet and the server response the document shows for it.
		{"spec/appendixB", "7878 0d 01 0353413532150362 0002 2d06 0d0a", "353413532150362", "7878 05 01 0002 eb47 0d0a"},
		// GS21: 0x0D length login, terminal ID only - no type identifier, no timezone.
		{"gs21", "7878 0d 01 0865948050016579 004b e935 0d0a", "865948050016579", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := &TR06Protocol{}
			ack, skip, err := p.Login(bufio.NewReader(bytes.NewReader(mustHex(t, c.frame))))
			require.NoError(t, err)
			assert.Equal(t, 0, skip)
			assert.Equal(t, c.wantIMEI, p.GetDeviceID(), "terminal ID is 8 BCD bytes holding a 15 digit IMEI (§5.1.1.4)")
			if c.wantAck != "" {
				assert.Equal(t, strings.ReplaceAll(c.wantAck, " ", ""), hex.EncodeToString(ack),
					"server response is 7878 05 <protocol> <serial> <crc> 0d0a (§5.1.2)")
			}
		})
	}
}

func TestLocationDataPacketSpec(t *testing.T) {
	// TR06 §5.2.2 / Appendix B location data packet (0x12).
	packet := parseFrame(t, "7878 1f 12 0b081d112e10 cf 027ac7eb 0c465849 00 148f 01cc 00 287d 001fb8 0003 8081 0d0a")
	require.Equal(t, MessageType(MSG_PositioningData), packet.MessageType)
	info, ok := packet.Information.(*PositioningInformation)
	require.True(t, ok)

	assert.Equal(t, time.Date(2011, 8, 29, 17, 46, 16, 0, time.UTC), info.GpsInformation.Timestamp.UTC())
	assert.Equal(t, uint8(12), info.GpsInformation.GPSInfoLength)
	// The satellite byte is 0xCF: length 12, 15 satellites. §5.2.2's "explain" row prints
	// 0xCC for it, but only 0xCF reproduces the document's own check bit 0x8081.
	assert.Equal(t, uint8(15), info.GpsInformation.NumberOfSatellites)
	assert.InDelta(t, 23.111668, info.GpsInformation.Latitude, 1e-4)
	assert.InDelta(t, 114.409285, info.GpsInformation.Longitude, 1e-4)
	assert.Equal(t, uint8(0), info.GpsInformation.Speed)
	assert.Equal(t, uint16(143), info.GpsInformation.Course.Degree)
	assert.True(t, info.GpsInformation.Course.Positioned)
	assert.True(t, info.GpsInformation.Course.Latitude, "north latitude")
	assert.False(t, info.GpsInformation.Course.Longitude, "east longitude")
	assert.True(t, info.GpsInformation.Course.IsRealtime, "bit5 = 0, real time GPS")
	assert.Equal(t, uint16(460), info.LBSInfo.MCC)
	assert.Equal(t, uint8(0), info.LBSInfo.MNC)
	assert.Equal(t, uint16(0x287d), info.LBSInfo.LAC)
	assert.Equal(t, [3]byte{0x00, 0x1f, 0xb8}, info.LBSInfo.CellID)
	assert.Equal(t, uint16(3), packet.InformationSerialNumber)
}

func TestLocationDataPacketGS21(t *testing.T) {
	// Real GS21 frame, moving vehicle: 10 satellites, 1 km/h, course 291.
	packet := parseFrame(t, "7878 1f 12 1a080d142131 ca 0162ad19 0855849f 01 d523 0194 2d 61f2 003df7 00e3 0d32 0d0a")
	info, ok := packet.Information.(*PositioningInformation)
	require.True(t, ok)

	assert.Equal(t, time.Date(2026, 8, 13, 20, 33, 49, 0, time.UTC), info.GpsInformation.Timestamp.UTC(),
		"the device's Date Time digits, read in the configured device timezone (UTC by default)")
	assert.Equal(t, uint8(10), info.GpsInformation.NumberOfSatellites)
	assert.InDelta(t, 12.913365, info.GpsInformation.Latitude, 1e-4)
	assert.InDelta(t, 77.679022, info.GpsInformation.Longitude, 1e-4)
	assert.Equal(t, uint8(1), info.GpsInformation.Speed)
	assert.Equal(t, uint16(291), info.GpsInformation.Course.Degree)
	assert.Equal(t, uint16(404), info.LBSInfo.MCC, "India")
	assert.Equal(t, uint8(45), info.LBSInfo.MNC)
	assert.Equal(t, uint16(0x61f2), info.LBSInfo.LAC)
	assert.Equal(t, [3]byte{0x00, 0x3d, 0xf7}, info.LBSInfo.CellID)

	// §5.2 puts no terminal status in the location packet, so ignition must stay unset.
	status := packet.ToProtobufDeviceStatus("865948050016579", types.DeviceType_WANWAY)
	assert.Nil(t, status.VehicleStatus.Ignition, "0x12 carries no ACC bit - nothing may be reported")
	assert.Equal(t, int32(10), status.Position.Satellites)
	assert.InDelta(t, float32(291), status.Position.Course, 0.001)
	assert.Equal(t, int64(1786653229), status.Timestamp.Seconds)
}

func TestAlarmPacketSpec(t *testing.T) {
	// TR06 §5.3.1.21 alarm packet (0x16): GPS + LBS + status, with the LBS Length byte.
	packet := parseFrame(t, "7878 25 16 0b0b0f0e241d cf 027ac887 0c4657e6 00 1402 09 01cc 00 287d 001f72 65 06 04 0101 0036 56a4 0d0a")
	require.Equal(t, MessageType(MSG_GPS_LBS_StatusData), packet.MessageType)
	info, ok := packet.Information.(*AlarmInformation)
	require.True(t, ok)

	assert.Equal(t, time.Date(2011, 11, 15, 14, 36, 29, 0, time.UTC), info.GpsInformation.Timestamp.UTC())
	assert.Equal(t, uint8(15), info.GpsInformation.NumberOfSatellites)
	assert.InDelta(t, 23.111755, info.GpsInformation.Latitude, 1e-4)
	assert.InDelta(t, 114.409230, info.GpsInformation.Longitude, 1e-4)
	assert.Equal(t, uint16(2), info.GpsInformation.Course.Degree)
	assert.Equal(t, uint16(460), info.LBSInformation.MCC)
	assert.Equal(t, uint16(0x287d), info.LBSInformation.LAC)
	assert.Equal(t, [3]byte{0x00, 0x1f, 0x72}, info.LBSInformation.CellID)

	// Terminal Information 0x65 = 0110 0101 (§5.3.1.14).
	term := info.StatusInformation.TerminalInformation
	assert.True(t, term.OilElectricityConnected, "bit7 = 0 means connected")
	assert.True(t, term.GPSSignalAvailable, "bit6 = 1, GPS tracking on")
	assert.Equal(t, AlarmType(AL_SOSDistress), term.AlarmType, "bit5..3 = 100, SOS")
	assert.True(t, term.Charging, "bit2 = 1, charge on")
	assert.False(t, term.ACCHigh, "bit1 = 0, ACC low")
	assert.True(t, term.Armed, "bit0 = 1, defense activated")

	assert.Equal(t, BatteryLevel(VL_BatteryFull), info.StatusInformation.BatteryLevel, "voltage level 6")
	assert.Equal(t, GSMSignalStrength(GSM_StrongSignal), info.StatusInformation.GSMSignalStrength, "0x04 strong")
	assert.Equal(t, AlarmValue(ALV_SOS), info.StatusInformation.Alarm)
	assert.Equal(t, Language(LANG_Chinese), info.StatusInformation.Language)

	status := packet.ToProtobufDeviceStatus("353413532150362", types.DeviceType_WANWAY)
	require.NotNil(t, status.VehicleStatus.Ignition)
	assert.False(t, *status.VehicleStatus.Ignition, "ACC low")
	assert.Equal(t, int32(100), status.BatteryLevel)
	assert.Equal(t, int32(4), status.GsmNetwork)
	assert.Equal(t, int32(15), status.Position.Satellites)
}

func TestAlarmPacketGS21(t *testing.T) {
	// Information content of a real GS21 0x16 packet (protocol byte + 32 byte body), fed
	// through parsePacketData - which is exactly what parsePacket hands the framed body to.
	packet := parseBody(t, "16 1a080d132f13 c7 0162673e 085637b8 00 145a 09 0194 2d 6208 005b38 40 04 03 ff02")
	require.Equal(t, MessageType(MSG_GPS_LBS_StatusData), packet.MessageType)

	info, ok := packet.Information.(*AlarmInformation)
	require.True(t, ok)
	assert.Equal(t, time.Date(2026, 8, 13, 19, 47, 19, 0, time.UTC), info.GpsInformation.Timestamp.UTC())
	assert.Equal(t, uint8(7), info.GpsInformation.NumberOfSatellites)
	assert.InDelta(t, 12.903430, info.GpsInformation.Latitude, 1e-4)
	assert.InDelta(t, 77.704493, info.GpsInformation.Longitude, 1e-4)
	assert.Equal(t, uint16(90), info.GpsInformation.Course.Degree)
	assert.True(t, info.GpsInformation.Course.Positioned)
	assert.Equal(t, uint16(404), info.LBSInformation.MCC)
	assert.Equal(t, uint8(45), info.LBSInformation.MNC)
	assert.Equal(t, uint16(0x6208), info.LBSInformation.LAC)

	// Terminal Information 0x40: GPS tracking on, ACC low, oil and electricity connected.
	term := info.StatusInformation.TerminalInformation
	assert.True(t, term.GPSSignalAvailable)
	assert.False(t, term.ACCHigh)
	assert.True(t, term.OilElectricityConnected)
	assert.False(t, term.Charging)
	assert.Equal(t, BatteryLevel(VL_BatteryMedium), info.StatusInformation.BatteryLevel)
	assert.Equal(t, GSMSignalStrength(GSM_GoodSignal), info.StatusInformation.GSMSignalStrength)
	assert.Equal(t, Language(LANG_English), info.StatusInformation.Language)
}

func TestHeartbeatPacket(t *testing.T) {
	cases := []struct {
		name        string
		frame       string
		wantACCHigh bool
		wantVoltage BatteryLevel
		wantGSM     GSMSignalStrength
	}{
		// TR06 Appendix B status packet: terminal 0x44, voltage 0x01, GSM 0x04.
		{"spec/appendixB", "7878 0a 13 44 01 04 0001 0005 0845 0d0a", false, VL_BatteryExtremelyLow, GSM_StrongSignal},
		// Real GS21: terminal 0x42 = GPS tracking on + ACC high.
		{"gs21/accHigh", "7878 0a 13 42 04 03 0002 00e2 0806 0d0a", true, VL_BatteryMedium, GSM_GoodSignal},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			packet := parseFrame(t, c.frame)
			require.Equal(t, MessageType(MSG_HeartbeatData), packet.MessageType)
			hb, ok := packet.Information.(*HeartbeatData)
			require.True(t, ok, "heartbeat must reach the proto mapper as *HeartbeatData")

			assert.Equal(t, c.wantACCHigh, hb.TerminalInformation.ACCHigh)
			assert.Equal(t, c.wantVoltage, hb.BatteryLevel)
			assert.Equal(t, c.wantGSM, hb.GSMSignalStrength)

			status := packet.ToProtobufDeviceStatus("865948050016579", types.DeviceType_WANWAY)
			require.NotNil(t, status.VehicleStatus.Ignition, "0x13 carries the ACC bit")
			assert.Equal(t, c.wantACCHigh, *status.VehicleStatus.Ignition)
			assert.Equal(t, int32(c.wantGSM), status.GsmNetwork)
			assert.Equal(t, resolveBatteryLevel(int32(c.wantVoltage)), status.BatteryLevel)
		})
	}
}

func TestTerminalInformationBits(t *testing.T) {
	p := &TR06Protocol{}

	// §5.3.1.14 worked example: 0x44 = oil and electricity connected, GPS tracking on,
	// normal (no alarm), charge on, ACC low, defense deactivated.
	term, err := p.parseTerminalInfoFromByte(0x44)
	require.NoError(t, err)
	assert.True(t, term.OilElectricityConnected)
	assert.True(t, term.GPSSignalAvailable)
	assert.Equal(t, AlarmType(AL_Normal), term.AlarmType)
	assert.True(t, term.Charging)
	assert.False(t, term.ACCHigh)
	assert.False(t, term.Armed)

	// 0x4B = 0100 1011 -> defense activated, ACC high, shock alarm, GPS tracking on.
	term, err = p.parseTerminalInfoFromByte(0x4B)
	require.NoError(t, err)
	assert.True(t, term.Armed)
	assert.True(t, term.ACCHigh)
	assert.Equal(t, AlarmType(AL_Vibration), term.AlarmType)
	assert.True(t, term.GPSSignalAvailable)

	// The byte the GS21 actually sends.
	term, err = p.parseTerminalInfoFromByte(0x42)
	require.NoError(t, err)
	assert.True(t, term.ACCHigh, "GS21 reports ACC high in this byte")
	assert.False(t, term.Charging)
	assert.False(t, term.Armed)

	term, err = p.parseTerminalInfoFromByte(0x80)
	require.NoError(t, err)
	assert.False(t, term.OilElectricityConnected, "bit7 = 1 means oil and electricity disconnected")
}

func TestHemisphereFromStatusBits(t *testing.T) {
	// §5.2.1.6/7 transmit unsigned magnitudes; the hemisphere is in the Course/Status bits.
	// Same coordinates as the spec location packet with the flags flipped to south + west
	// (status byte 0x14 -> 0x18: bit2 north cleared, bit3 west set).
	packet := parseBody(t, "12 0b081d112e10 cf 027ac7eb 0c465849 00 188f 01cc 00 287d 001fb8")
	info := packet.Information.(*PositioningInformation)
	assert.InDelta(t, -23.111668, info.GpsInformation.Latitude, 1e-4, "south latitude is negative")
	assert.InDelta(t, -114.409285, info.GpsInformation.Longitude, 1e-4, "west longitude is negative")
}

func TestUnsupportedProtocolNumberIsSkippedNotFatal(t *testing.T) {
	// TR06 §4.3 defines 0x01, 0x12, 0x13, 0x15, 0x16, 0x1A and 0x80. Anything else must be
	// discarded without failing the packet, so the terminal's link survives (§iii.7).
	packet := parseBody(t, "15 01020304")
	assert.Nil(t, packet.Information)
	assert.Equal(t, MessageType(0x15), packet.MessageType, "TR06 §4.3 string information packet")
}

func TestStartBitAndLengthGuards(t *testing.T) {
	p := &TR06Protocol{}
	// TR06 §4.1: the start bit is fixed at 0x7878, there is no 0x7979 frame.
	assert.False(t, p.IsValidHeader(bufio.NewReader(bytes.NewReader(mustHex(t, "7979")))))
	assert.True(t, p.IsValidHeader(bufio.NewReader(bytes.NewReader(mustHex(t, "7878")))))

	_, err := p.parsePacket(bufio.NewReader(bytes.NewReader(mustHex(t, "7979 0005 01 0001 0d0a"))))
	assert.Error(t, err, "0x7979 must be rejected, never read as a zero length frame")

	_, err = p.parsePacket(bufio.NewReader(bytes.NewReader(mustHex(t, "7878 00 13 0d0a"))))
	assert.Error(t, err, "a length below the 5 byte minimum must be rejected")
}

func TestDeviceTimeZoneOffset(t *testing.T) {
	// TR06 gives no timezone for Date Time, so the offset is configuration, not inference.
	assert.Equal(t, time.UTC, loadDeviceTimeZone(""), "default is unchanged behaviour")
	assert.Equal(t, time.UTC, loadDeviceTimeZone("garbage"))

	ist := loadDeviceTimeZone("+05:30")
	_, offset := time.Now().In(ist).Zone()
	assert.Equal(t, 19800, offset)

	// The GS21 frame above holds the digits 2026-08-13 20:33:49. Read as +05:30 that is
	// 15:03:49Z - the second the receiver logged it.
	assert.Equal(t,
		time.Date(2026, 8, 13, 15, 3, 49, 0, time.UTC),
		time.Date(2026, 8, 13, 20, 33, 49, 0, ist).UTC())
}
