package wanway

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"io"
	"testing"
	"time"

	"github.com/404minds/avl-receiver/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestParseLoginInformation(t *testing.T) {
	imeiStr := "0123456789012345"
	typeIdentifierStr := "0518"
	timezoneStr := "4dd8"
	data, _ := hex.DecodeString(imeiStr + typeIdentifierStr + timezoneStr)

	p := WanwayProtocol{}
	reader := bufio.NewReader(bytes.NewReader(data))
	parsedInfo, err := p.parseLoginInformation(reader)

	if assert.NoError(t, err, "parseLoginInformation should succeed") {
		if assert.IsType(t, &WanwayLoginData{}, parsedInfo, "loginInfo should be of type WanwayLoginInformation") {
			loginInfo := parsedInfo.(*WanwayLoginData)
			assert.Equal(t, "123456789012345", loginInfo.TerminalID, "imei number should match")
			assert.Equal(t, time.FixedZone("", -(12*60*60+45*60)), loginInfo.Timezone, "timezone should match")
		}
	}
}

func TestIsWanwayHeader(t *testing.T) {
	cases := []struct {
		data     []byte
		expected bool
	}{
		{[]byte{0x78, 0x78}, true},
		{[]byte{0x79, 0x79}, true},
		{[]byte{0x78, 0x79}, false},
		{[]byte{0x79, 0x78}, false},
	}

	p := WanwayProtocol{}
	for _, c := range cases {
		reader := bufio.NewReader(bytes.NewReader(c.data))
		assert.Equal(t, c.expected, p.IsWanwayHeader(reader), "header should be detected properly")
	}
}

func TestParseWanwaypacket(t *testing.T) {
	startBit := "7878"
	packetLength := "11" // 17
	messageType := "01"
	imei := "0123456789012345"
	typeIdentifier := "0518"
	timezone := "4dd8"
	informationNumber := "0001"
	crc := "e2c0"
	stopBits := "0d0a"

	data, _ := hex.DecodeString(startBit + packetLength + messageType + imei + typeIdentifier + timezone + informationNumber + crc + stopBits)
	reader := bufio.NewReader(bytes.NewReader(data))

	p := WanwayProtocol{}
	packet, err := p.parseWanwayPacket(reader)

	if assert.NoError(t, err, "parseWanwayPacket should succeed") {
		assert.Equal(t, uint16(0x7878), packet.StartBit, "start bit should match")
		assert.Equal(t, uint16(0x0d0a), packet.StopBits, "start bit should match")
		assert.Equal(t, int8(17), packet.PacketLength, "packet length should match")
		assert.Equal(t, MSG_LoginData, packet.MessageType, "message type should match")
		assert.Equal(t, uint16(0x0001), packet.InformationSerialNumber, "information serial number should match")
		assert.Equal(t, uint16(0xe2c0), packet.Crc, "crc should match")

		if assert.IsType(t, &WanwayLoginData{}, packet.Information, "packet information should be of type WanwayLoginInformation") {
			loginInfo := packet.Information.(*WanwayLoginData)
			assert.Equal(t, "123456789012345", loginInfo.TerminalID, "imei number should match")
			assert.Equal(t, time.FixedZone("", -(12*60*60+45*60)), loginInfo.Timezone, "timezone should match")
		}
	}
}

func TestWanwayLoginMessage(t *testing.T) {
	startBit := "7878"
	packetLength := "11" // 17
	messageType := "01"
	imei := "0123456789012345"
	typeIdentifier := "0518"
	timezone := "4dd8"
	informationNumber := "0001"
	crc := "e2c0"
	stopBits := "0d0a"

	loginmsg, _ := hex.DecodeString(startBit + packetLength + messageType + imei + typeIdentifier + timezone + informationNumber + crc + stopBits)
	reader := bufio.NewReader(bytes.NewReader(loginmsg))

	var writeBuffer bytes.Buffer
	writer := bufio.NewWriter(&writeBuffer)
	var c chan types.DeviceStatus

	p := WanwayProtocol{}
	err := p.ConsumeStream(reader, writer, c)
	assert.ErrorIs(t, err, io.EOF)

	expectedResponse := startBit + packetLength + messageType + informationNumber + crc + stopBits
	assert.Equal(t, expectedResponse, hex.EncodeToString(writeBuffer.Bytes()))
}
