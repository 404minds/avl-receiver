package teltonika

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTeltonikaLogin(t *testing.T) {
	buf, _ := hex.DecodeString("000F333536333037303433373231353739")
	randBytes := make([]byte, 100)
	rand.Read(randBytes)
	buf = append(buf, randBytes...) // append some random data to mimic some continuous data stream
	reader := bufio.NewReader(bytes.NewReader(buf))

	teltonika := TeltonikaProtocol{}

	expectedImei := "356307043721579"
	imei, bytesConsumed, _ := teltonika.peekImei(reader)
	assert.Equal(t, expectedImei, imei, "Teltonika peekImei failed")
	assert.Equal(t, 17, bytesConsumed, "Bytes consumed")

	newBuf, _ := reader.Peek(len(buf))
	assert.Equal(t, buf, newBuf, "Teltonika login should not consume any bytes")

	ack, bytesConsumed, err := teltonika.Login(reader)
	if assert.NoError(t, err, "Teltonika login failed") {
		assert.Equal(t, []byte{0x01}, ack, "Teltonika login failed")
		assert.Equal(t, 17, bytesConsumed, "Bytes consumed")
	}

	newBuf, _ = reader.Peek(len(buf))
	assert.Equal(t, buf, newBuf, "Teltonika login should not consume any bytes")
}

func TestDataPacketParsing(t *testing.T) {
}

func TestGpsParsing(t *testing.T) {
	type testCase struct {
		Bytes    string
		Expected TeltonikaGpsElement
	}
	testCases := []testCase{
		{
			Bytes: "0F0EC760209A6B0000620000060000",
			Expected: TeltonikaGpsElement{
				Longitude:  252626784,
				Latitude:   546990848,
				Altitude:   98,
				Angle:      0,
				Satellites: 6,
				Speed:      0,
			},
		},
		{
			Bytes: "0F0EB790209A778000AB010C050000",
			Expected: TeltonikaGpsElement{
				Longitude:  252622736,
				Latitude:   546994048,
				Altitude:   171,
				Angle:      268,
				Satellites: 5,
				Speed:      0,
			},
		},
	}

	teltonika := TeltonikaProtocol{Imei: "something"}
	for _, testCase := range testCases {
		buf, _ := hex.DecodeString(testCase.Bytes)
		reader := bufio.NewReader(bytes.NewReader(buf))
		gpsElement, err := teltonika.parseGpsElement(reader)

		expected := testCase.Expected
		if assert.NoError(t, err, "Teltonika parseGpsElement failed") {
			assert.Equal(t, expected.Longitude, gpsElement.Longitude, "incorrect longitude")
			assert.Equal(t, expected.Latitude, gpsElement.Latitude, "incorrect latitude")
			assert.Equal(t, expected.Altitude, gpsElement.Altitude, "incorrect altitude")
			assert.Equal(t, expected.Angle, gpsElement.Angle, "incorrect angle")
			assert.Equal(t, expected.Satellites, gpsElement.Satellites, "incorrect satellites")
			assert.Equal(t, expected.Speed, gpsElement.Speed, "incorrect speed")
		}
	}
}

func TestIOElementParsing(t *testing.T) {
	type testCase struct {
		Bytes    string
		Expected TeltonikaIOElement
	}

	testCases := []testCase{
		{
			Bytes: "00170A010002000300B300B4004501F00150041503C80008B50012B6000A423024180000CD0386CE0001431057440000044600000112C700000000F10000601A4800000000014E0000000000000000",
			Expected: TeltonikaIOElement{
				EventID:       0,
				NumProperties: 23,
				Properties1B: map[TeltonikaIOProperty]uint8{
					DigitalInput1:  0,
					DigitalInput2:  0,
					DigitalInput3:  0,
					DigitalOutput1: 0,
					DigitalOutput2: 0,
					GPSPower:       1,
					MovementSensor: 1,
					WorkingMode:    4,
					GSMSignal:      3,
					SleepMode:      0,
				},
				Properties2B: map[TeltonikaIOProperty]uint16{
					GPSPDOP:         18,
					GPSHDOP:         10,
					ExternalVoltage: 12324,
					Speed:           0,
					CellID:          902,
					AreaCode:        1,
					BatteryVoltage:  4183,
					BatteryCurrent:  0,
				},
				Properties4B: map[TeltonikaIOProperty]uint32{
					PCBTemperature:    274,
					OdometerValue:     0,
					GSMOperator:       24602,
					DallasTemperature: 0,
				},
				Properties8B: map[TeltonikaIOProperty]uint64{
					IButtonID: 0,
				},
			},
		},
		{
			Bytes: "000000000000",
			Expected: TeltonikaIOElement{
				EventID:       0,
				NumProperties: 0,
				Properties1B:  map[TeltonikaIOProperty]uint8{},
				Properties2B:  map[TeltonikaIOProperty]uint16{},
				Properties4B:  map[TeltonikaIOProperty]uint32{},
				Properties8B:  map[TeltonikaIOProperty]uint64{},
			},
		},
	}

	teltonika := TeltonikaProtocol{Imei: "something"}
	for _, testCase := range testCases {
		buf, _ := hex.DecodeString(testCase.Bytes)
		reader := bufio.NewReader(bytes.NewReader(buf))

		ioElement, err := teltonika.parseIOElements(reader)
		if assert.NoError(t, err, "Teltonika parseIOElements failed") {
			expected := testCase.Expected
			assert.Equal(t, expected.EventID, ioElement.EventID, "incorrect event id")
			assert.Equal(t, expected.NumProperties, ioElement.NumProperties, "incorrect num properties")
			assert.Equal(t, expected.Properties1B, ioElement.Properties1B, "incorrect 1B properties")
			assert.Equal(t, expected.Properties2B, ioElement.Properties2B, "incorrect 2B properties")
			assert.Equal(t, expected.Properties4B, ioElement.Properties4B, "incorrect 4B properties")
			assert.Equal(t, expected.Properties8B, ioElement.Properties8B, "incorrect 8B properties")
		}
	}
}
