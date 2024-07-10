package gt06

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseLoginInformation(t *testing.T) {
	imeiStr := "0123456789012345"
	typeIdentifierStr := "0518"
	timezoneStr := "4dd8"
	data, _ := hex.DecodeString(imeiStr + typeIdentifierStr + timezoneStr)

	p := GT06Protocol{}
	reader := bufio.NewReader(bytes.NewReader(data))
	parsedInfo, err := p.parseLoginInformation(reader)

	if assert.NoError(t, err, "parseLoginInformation should succeed") {
		if assert.IsType(t, &LoginData{}, parsedInfo, "loginInfo should be of type WanwayLoginInformation") {
			loginInfo := parsedInfo.(*LoginData)
			assert.Equal(t, "123456789012345", loginInfo.TerminalID, "imei number should match")
			assert.Equal(t, time.FixedZone("", -(12*60*60+45*60)), loginInfo.Timezone, "timezone should match")
		}
	}
}

func TestHeaderDetection(t *testing.T) {
	cases := []struct {
		data     []byte
		expected bool
	}{
		{[]byte{0x78, 0x78}, true},
		{[]byte{0x79, 0x79}, true},
		{[]byte{0x78, 0x79}, false},
		{[]byte{0x79, 0x78}, false},
	}

	p := GT06Protocol{}
	for _, c := range cases {
		reader := bufio.NewReader(bytes.NewReader(c.data))
		assert.Equal(t, c.expected, p.IsValidHeader(reader), "header should be detected properly")
	}
}

func TestPacketParsing(t *testing.T) {
	startBit := "7878"
	packetLength := "11" // 17
	messageType := "01"
	imei := "0123456789012345"
	typeIdentifier := "0518"
	timezone := "4dd8"
	informationNumber := "0001"
	crc := "cb97"
	stopBits := "0d0a"

	data, _ := hex.DecodeString(startBit + packetLength + messageType + imei + typeIdentifier + timezone + informationNumber + crc + stopBits)
	reader := bufio.NewReader(bytes.NewReader(data))

	p := GT06Protocol{}
	packet, err := p.parsePacket(reader)

	if assert.NoError(t, err, "parsePacket should succeed") {
		assert.Equal(t, uint16(0x7878), packet.StartBit, "start bit should match")
		assert.Equal(t, uint16(0x0d0a), packet.StopBits, "start bit should match")
		assert.Equal(t, int8(17), packet.PacketLength, "packet length should match")
		assert.Equal(t, MSG_LoginData, packet.MessageType, "message type should match")
		assert.Equal(t, uint16(0x0001), packet.InformationSerialNumber, "information serial number should match")
		assert.Equal(t, uint16(0xcb97), packet.Crc, "crc should match")

		if assert.IsType(t, &LoginData{}, packet.Information, "packet information should be of type WanwayLoginInformation") {
			loginInfo := packet.Information.(*LoginData)
			assert.Equal(t, "123456789012345", loginInfo.TerminalID, "imei number should match")
			assert.Equal(t, time.FixedZone("", -(12*60*60+45*60)), loginInfo.Timezone, "timezone should match")
		}
	}
}

func TestParseLoginMessage(t *testing.T) {
	startBit := "7878"
	packetLength := "11" // 17
	messageType := "01"
	imei := "0123456789012345"
	typeIdentifier := "0518"
	timezone := "4dd8"
	informationNumber := "0001"
	crc := "cb97"
	stopBits := "0d0a"

	loginmsg, _ := hex.DecodeString(startBit + packetLength + messageType + imei + typeIdentifier + timezone + informationNumber + crc + stopBits)
	reader := bufio.NewReader(bytes.NewReader(loginmsg))

	p := GT06Protocol{}
	ack, bytesToSkip, err := p.Login(reader)

	assert.NoError(t, err, "Login should succeed")
	assert.Equal(t, 0, bytesToSkip, "bytesToSkip should be 17")

	assert.Equal(t, imei[1:], p.GetDeviceID(), "device identifier should match")

	expectedResponse := startBit + packetLength + messageType + informationNumber + crc + stopBits
	assert.Equal(t, expectedResponse, hex.EncodeToString(ack))
}

func TestParseHeartbeatPacket(t *testing.T) {
	bytestr := strings.ReplaceAll("78 78 0A 13 40 04 04 00 01 00 0F DC EE 0D 0A", " ", "")
	data, _ := hex.DecodeString(bytestr)

	p := GT06Protocol{}

	packet, err := p.parsePacket(bufio.NewReader(bytes.NewReader(data)))
	assert.NoError(t, err, "should parse heartbeat packet")

	assert.Equal(t, uint16(0x7878), packet.StartBit, "start bits should match")
	assert.Equal(t, uint16(0x0d0a), packet.StopBits, "stop bits should match")
	assert.Equal(t, int8(10), packet.PacketLength, "packet length should match")
	assert.Equal(t, MessageType(MSG_HeartbeatData), packet.MessageType, "message type should match")
	assert.Equal(t, uint16(0x000f), packet.InformationSerialNumber, "information serial number should match")
	assert.Equal(t, uint16(0xdcee), packet.Crc, "crc should match")

	heartbeatData := packet.Information.(HeartbeatData)
	assert.Equal(t, false, heartbeatData.TerminalInformation.OilElectricityConnected, "termInfo: oil electricity connected should match")
	assert.Equal(t, true, heartbeatData.TerminalInformation.GPSSignalAvailable, "termInfo: gps signal available")
	assert.Equal(t, false, heartbeatData.TerminalInformation.Charging, "termInfo: charging should match")
	assert.Equal(t, false, heartbeatData.TerminalInformation.ACCHigh, "termInfo: acc high should match")
	assert.Equal(t, false, heartbeatData.TerminalInformation.Armed, "termInfo: armed should match")

	assert.Equal(t, BatteryLevel(VL_BatteryMedium), heartbeatData.BatteryLevel, "battery level should match")
	assert.Equal(t, GSMSignalStrength(GSM_StrongSignal), heartbeatData.GSMSignalStrength, "gsm signal strength should match")
	assert.Equal(t, uint16(0x0001), heartbeatData.ExtendedPortStatus, "alarm status should match")
}

func TestParseGpsLocationPacket(t *testing.T) {
	bytestr := strings.ReplaceAll("78 78 22 22 0F 0C 1D 02 33 05 C9 02 7A C8 18 0C 46 58 60 00 14 00 01 CC 00 28 7D 00 1F 71 00 00 01 00 08 20 86 0D 0A", " ", "")
	data, _ := hex.DecodeString(bytestr)

	p := GT06Protocol{LoginInformation: &LoginData{Timezone: time.UTC}}

	packet, err := p.parsePacket(bufio.NewReader(bytes.NewReader(data)))
	if err != nil {
		t.Fatalf("should parse gps location packet, but got error: %v", err)
	}

	if packet == nil {
		t.Fatalf("packet should not be nil")
	}

	assert.Equal(t, uint16(0x7878), packet.StartBit, "start bits should match")
	assert.Equal(t, uint16(0x0d0a), packet.StopBits, "stop bits should match")
	assert.Equal(t, int8(34), packet.PacketLength, "packet length should match")
	assert.Equal(t, MessageType(MSG_PositioningData), packet.MessageType, "message type should match")
	assert.Equal(t, uint16(0x0008), packet.InformationSerialNumber, "information serial number should match")
	assert.Equal(t, uint16(0x2086), packet.Crc, "crc should match")

	gpsData, ok := packet.Information.(GPSInformation)
	if !ok {
		t.Fatalf("packet information should be of type GPSInformation")
	}

	assert.Equal(t, time.Date(16, 12, 29, 2, 51, 5, 0, time.UTC), gpsData.Timestamp, "timestamp should match")
	assert.Equal(t, uint(8), gpsData.GPSInfoLength, "gps info length should match")
	assert.Equal(t, uint(9), gpsData.NumberOfSatellites, "number of satellites should match")
	assert.Equal(t, float32(41601048/1800000), gpsData.Latitude, "latitude should match")
	assert.Equal(t, float32(52719804416/1800000), gpsData.Longitude, "longitude should match")
}
