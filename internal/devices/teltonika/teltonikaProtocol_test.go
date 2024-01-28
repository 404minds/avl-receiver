package teltonika

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
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
	buf, _ := hex.DecodeString("00000000000000A608030000013FEB40E0B2000F0EC760209A6B000062000006000000170A010002000300B300B4004501F00150041503C80008B50012B6000A423024180000CD0386CE0001431057440000044600000112C700000000F10000601A4800000000014E00000000000000000000013F14A1D1CE000F0EB790209A778000AB010C0500000000000000000000013F1498A63A000F0EB790209A77800095010C0400000000000000000300003390")
	reader := bufio.NewReader(bytes.NewReader(buf))

	var writeBuffer bytes.Buffer
	writer := bufio.NewWriter(&writeBuffer)
	storeProcessChan := make(chan interface{}, 200)

	teltonika := TeltonikaProtocol{Imei: "something"}

	err := teltonika.ConsumeStream(reader, writer, storeProcessChan)
	assert.ErrorIs(t, err, io.EOF)

	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x03}, writeBuffer.Bytes(), "Incorrect ack from consume data")

	assert.Len(t, storeProcessChan, 3, "Incorrect number of records sent to store")

	firstRecord := (<-storeProcessChan).(TeltonikaRecord)
	assert.Equal(t, firstRecord.IMEI, "something", "Incorrect IMEI")
	assert.Equal(t, firstRecord.Record.Priority, uint8(0), "Incorrect priority")
	assert.Equal(t, firstRecord.Record.Timestamp, uint64(1374041465010), "Incorrect timestamp")
	assert.NotNil(t, firstRecord.Record.GPSElement, "Incorrect gps element")
	assert.NotNil(t, firstRecord.Record.IOElement, "Incorrect gps element")
	assert.Equal(t, firstRecord.Record.IOElement.EventID, uint8(0), "Incorrect event id")
	assert.Equal(t, firstRecord.Record.IOElement.NumProperties, uint8(23), "Incorrect number of IO elements")

	secondRecord := (<-storeProcessChan).(TeltonikaRecord)
	assert.Equal(t, secondRecord.IMEI, "something", "Incorrect IMEI")
	assert.Equal(t, secondRecord.Record.Priority, uint8(0), "Incorrect priority")
	assert.Equal(t, secondRecord.Record.Timestamp, uint64(1370440716750), "Incorrect timestamp")
	assert.NotNil(t, secondRecord.Record.GPSElement, "Incorrect gps element")
	assert.NotNil(t, secondRecord.Record.IOElement, "Incorrect gps element")
	assert.Equal(t, secondRecord.Record.IOElement.EventID, uint8(0), "Incorrect event id")
	assert.Equal(t, secondRecord.Record.IOElement.NumProperties, uint8(0), "Incorrect number of IO elements")

	thirdRecord := (<-storeProcessChan).(TeltonikaRecord)
	assert.Equal(t, thirdRecord.IMEI, "something", "Incorrect IMEI")
	assert.Equal(t, thirdRecord.Record.Priority, uint8(0), "Incorrect priority")
	assert.Equal(t, thirdRecord.Record.Timestamp, uint64(1370440115770), "Incorrect timestamp")
	assert.NotNil(t, thirdRecord.Record.GPSElement, "Incorrect gps element")
	assert.NotNil(t, thirdRecord.Record.IOElement, "Incorrect gps element")
	assert.Equal(t, thirdRecord.Record.IOElement.EventID, uint8(0), "Incorrect event id")
	assert.Equal(t, thirdRecord.Record.IOElement.NumProperties, uint8(0), "Incorrect number of IO elements")
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
					TIO_DigitalInput1:  0,
					TIO_DigitalInput2:  0,
					TIO_DigitalInput3:  0,
					TIO_DigitalOutput1: 0,
					TIO_DigitalOutput2: 0,
					TIO_GPSPower:       1,
					TIO_MovementSensor: 1,
					TIO_WorkingMode:    4,
					TIO_GSMSignal:      3,
					TIO_SleepMode:      0,
				},
				Properties2B: map[TeltonikaIOProperty]uint16{
					TIO_GPSPDOP:         18,
					TIO_GPSHDOP:         10,
					TIO_ExternalVoltage: 12324,
					TIO_Speed:           0,
					TIO_CellID:          902,
					TIO_AreaCode:        1,
					TIO_BatteryVoltage:  4183,
					TIO_BatteryCurrent:  0,
				},
				Properties4B: map[TeltonikaIOProperty]uint32{
					TIO_PCBTemperature:    274,
					TIO_OdometerValue:     0,
					TIO_GSMOperator:       24602,
					TIO_DallasTemperature: 0,
				},
				Properties8B: map[TeltonikaIOProperty]uint64{
					TIO_IButtonID: 0,
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

func Test1BIOElementParsing(t *testing.T) {
	type testCase struct {
		Bytes    string
		Expected map[TeltonikaIOProperty]uint8
	}

	testCases := []testCase{
		{
			Bytes: "0A010002000300B300B4004501F00150041503C800",
			Expected: map[TeltonikaIOProperty]uint8{
				TIO_DigitalInput1:  0,
				TIO_DigitalInput2:  0,
				TIO_DigitalInput3:  0,
				TIO_DigitalOutput1: 0,
				TIO_DigitalOutput2: 0,
				TIO_GPSPower:       1,
				TIO_MovementSensor: 1,
				TIO_WorkingMode:    4,
				TIO_GSMSignal:      3,
				TIO_SleepMode:      0,
			},
		},
		{
			Bytes:    "00",
			Expected: map[TeltonikaIOProperty]uint8{},
		},
	}

	teltonika := TeltonikaProtocol{Imei: "generic-imei"}
	for _, testCase := range testCases {
		buf, _ := hex.DecodeString(testCase.Bytes)
		reader := bufio.NewReader(bytes.NewReader(buf))

		ioElement, err := teltonika.read1BProperties(reader)
		if assert.NoError(t, err, "read1BProperties failed") {
			assert.Equal(t, testCase.Expected, ioElement, "incorrect 1B properties")
		}
	}
}

func Test2BIOElementParsing(t *testing.T) {
	type testCase struct {
		Bytes    string
		Expected map[TeltonikaIOProperty]uint16
	}

	testCases := []testCase{
		{
			Bytes: "08B50012B6000A423024180000CD0386CE0001431057440000",
			Expected: map[TeltonikaIOProperty]uint16{
				TIO_GPSPDOP:         18,
				TIO_GPSHDOP:         10,
				TIO_ExternalVoltage: 12324,
				TIO_Speed:           0,
				TIO_CellID:          902,
				TIO_AreaCode:        1,
				TIO_BatteryVoltage:  4183,
				TIO_BatteryCurrent:  0,
			},
		},
		{
			Bytes:    "00",
			Expected: map[TeltonikaIOProperty]uint16{},
		},
	}

	teltonika := TeltonikaProtocol{Imei: "generic-imei"}
	for _, testCase := range testCases {
		buf, _ := hex.DecodeString(testCase.Bytes)
		reader := bufio.NewReader(bytes.NewReader(buf))

		ioElement, err := teltonika.read2BProperties(reader)
		if assert.NoError(t, err, "read2BProperties failed") {
			assert.Equal(t, testCase.Expected, ioElement, "incorrect 2B properties")
		}
	}
}

func Test4BIOElementParsing(t *testing.T) {
	type testCase struct {
		Bytes    string
		Expected map[TeltonikaIOProperty]uint32
	}

	testCases := []testCase{
		{
			Bytes: "044600000112C700000000F10000601A4800000000",
			Expected: map[TeltonikaIOProperty]uint32{
				TIO_PCBTemperature:    274,
				TIO_OdometerValue:     0,
				TIO_GSMOperator:       24602,
				TIO_DallasTemperature: 0,
			},
		},
		{
			Bytes:    "00",
			Expected: map[TeltonikaIOProperty]uint32{},
		},
	}

	teltonika := TeltonikaProtocol{Imei: "generic-imei"}
	for _, testCase := range testCases {
		buf, _ := hex.DecodeString(testCase.Bytes)
		reader := bufio.NewReader(bytes.NewReader(buf))

		ioElement, err := teltonika.read4BProperties(reader)
		if assert.NoError(t, err, "read4BProperties failed") {
			assert.Equal(t, testCase.Expected, ioElement, "incorrect 4B properties")
		}
	}
}

func Test16BIOElementParsing(t *testing.T) {
	type testCase struct {
		Bytes    string
		Expected map[TeltonikaIOProperty]uint64
	}

	testCases := []testCase{
		{
			Bytes: "014E0000000000000000",
			Expected: map[TeltonikaIOProperty]uint64{
				TIO_IButtonID: 0,
			},
		},
		{
			Bytes:    "00",
			Expected: map[TeltonikaIOProperty]uint64{},
		},
	}

	teltonika := TeltonikaProtocol{Imei: "generic-imei"}
	for _, testCase := range testCases {
		buf, _ := hex.DecodeString(testCase.Bytes)
		reader := bufio.NewReader(bytes.NewReader(buf))

		ioElement, err := teltonika.read8BProperties(reader)
		if assert.NoError(t, err, "read8BProperties failed") {
			assert.Equal(t, testCase.Expected, ioElement, "incorrect 8B properties")
		}
	}
}
