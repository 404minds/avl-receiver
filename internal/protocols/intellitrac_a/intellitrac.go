package intellitrac_a

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
)

var logger = configuredLogger.Logger

var (
	ErrInvalidHeartbeat    = errors.New("invalid heartbeat format")
	ErrUnsupportedMsgType  = errors.New("unsupported message type")
	ErrInvalidPositionData = errors.New("invalid position data")
	ErrChecksumMismatch    = errors.New("checksum mismatch")
	ErrInvalidLogin        = errors.New("invalid login")
)

func (t *IntelliTracAProtocol) GetDeviceType() types.DeviceType {
	return t.DeviceType
}

func (t *IntelliTracAProtocol) SetDeviceType(dt types.DeviceType) {
	t.DeviceType = dt
}

func (t *IntelliTracAProtocol) GetProtocolType() types.DeviceProtocolType {
	return types.DeviceProtocolType_INTELLITRAC_A
}

func (t *IntelliTracAProtocol) GetDeviceID() string {
	return t.Imei
}

func (t *IntelliTracAProtocol) Login(reader *bufio.Reader) ([]byte, int, error) {
	peeked, err := reader.Peek(4)
	if err != nil {
		return nil, 0, err
	}
	logger.Sugar().Infoln("peeked", peeked)

	var transactionID uint16
	var modemID uint64

	// Check if binary heartbeat (bytes 2-3: [0x00 0x02])
	if len(peeked) >= 4 && peeked[2] == 0x00 && peeked[3] == 0x02 {
		// Read full binary heartbeat
		data := make([]byte, BinaryHeartbeatSize)
		_, err := io.ReadFull(reader, data)
		if err != nil {
			return nil, 0, ErrInvalidHeartbeat
		}

		// Parse binary heartbeat according to page 12
		transactionID = binary.BigEndian.Uint16(data[0:2])
		// Message Encoding = data[2] (should be 0x00)
		// Message Type = data[3] (should be 0x02)
		modemID = binary.BigEndian.Uint64(data[4:12]) //make sure of 15 digits
		messageID := binary.BigEndian.Uint16(data[12:14])
		dataLength := binary.BigEndian.Uint16(data[14:16])

		logger.Sugar().Infof("Login - Modem ID: %d, Message ID: 0x%04X, Data Length: %d",
			modemID, messageID, dataLength)

		// Verify heartbeat message ID (0xAB)
		if messageID != 0xAB {
			logger.Sugar().Warnf("Expected heartbeat message ID 0xAB, got 0x%X", messageID)
		}

		// Verify data length (should be 6 for RTC data)
		if dataLength != 6 {
			logger.Sugar().Warnf("Expected data length 6, got %d", dataLength)
		}

		t.IsBinary = true
		logger.Sugar().Infof("Binary heartbeat - Transaction ID: %d, Modem ID: %d", transactionID, modemID)

	} else if len(peeked) >= 2 && peeked[0] == 0xFA && peeked[1] == 0xF8 {
		// ASCII heartbeat (page 11)
		data := make([]byte, ASCIIHeartbeatSize)
		_, err := io.ReadFull(reader, data)
		if err != nil {
			return nil, 0, ErrInvalidHeartbeat
		}

		// bytes 2–3 = Sequence ID (use as Transaction ID)
		transactionID = binary.BigEndian.Uint16(data[2:4])
		modemID = uint64(binary.BigEndian.Uint32(data[4:8]))
		t.IsBinary = false
		logger.Sugar().Infof("ASCII heartbeat - Transaction ID: %d, Modem ID: %d", transactionID, modemID)
	} else {
		return nil, 0, errs.ErrUnknownProtocol
	}

	t.Imei = padIMEI(fmt.Sprintf("%d", modemID))
	if !t.isImeiAuthorized(t.Imei) {
		return nil, 0, errs.ErrUnknownProtocol
	}

	// Form binary acknowledgment (page 10)
	ack := make([]byte, BinaryAckSize)
	binary.BigEndian.PutUint16(ack[0:2], transactionID)
	ack[2] = 0x00                                // Message Encoding: Binary Data
	ack[3] = 0x03                                // Message Type: Acknowledge
	binary.BigEndian.PutUint16(ack[4:6], 0x0000) // Status Code: Success

	return ack, 0, nil
}

func (t *IntelliTracAProtocol) ConsumeStream(reader *bufio.Reader, writer io.Writer, store store.Store) error {
	for {
		if t.IsBinary {
			err := t.consumeBinaryStream(reader, writer, store)
			if err != nil {
				return err
			}
		} else {
			err := t.consumeASCIIStream(reader, writer, store)
			if err != nil {
				return err
			}
		}
	}
}

func (t *IntelliTracAProtocol) consumeBinaryStream(reader *bufio.Reader, writer io.Writer, store store.Store) error {
	header := make([]byte, 4)

	if _, err := io.ReadFull(reader, header); err != nil {
		return err
	}
	logger.Sugar().Infoln("header", header)

	transactionID := binary.BigEndian.Uint16(header[0:2])
	msgEncoding := header[2]
	msgType := header[3]

	logger.Sugar().Infoln("msgEncoding", msgEncoding)
	logger.Sugar().Infoln("msgType", msgType)

	switch {
	case msgEncoding == MsgEncodingBinaryPos && msgType == MsgTypeAsync:
		return t.handleBinaryPosition(reader, writer, store, transactionID)
	case msgEncoding == MsgEncodingText && msgType == MsgTypeAsync:
		return t.handleTextMessage(reader, writer, store, transactionID)
	case msgEncoding == MsgEncodingATCommand && msgType == MsgTypeResponse:
		return t.handleATResponse(reader, writer, store, transactionID)
	default:
		return ErrUnsupportedMsgType
	}
}

func (t *IntelliTracAProtocol) handleBinaryPosition(reader *bufio.Reader, writer io.Writer, store store.Store, transactionID uint16) error {

	header := make([]byte, 12) // 8 bytes modemID + 4 bytes header
	if _, err := io.ReadFull(reader, header); err != nil {
		return err
	}
	imei := binary.BigEndian.Uint64(header[0:8])

	modemID := padIMEI(fmt.Sprintf("%d", imei))
	messageID := binary.BigEndian.Uint16(header[8:10])
	dataLen := binary.BigEndian.Uint16(header[10:12])

	logger.Sugar().Infof("Position - Modem ID: %d, Message ID: 0x%04X, Data Length: %d",
		modemID, messageID, dataLen)

	// Read position data
	data := make([]byte, dataLen)
	if _, err := io.ReadFull(reader, data); err != nil {
		return err
	}
	logger.Sugar().Infoln("data", data)

	t.handlePositionalData(data, modemID, dataLen, messageID, transactionID, writer, store)

	// Send acknowledgment
	ack := make([]byte, BinaryAckSize)
	binary.BigEndian.PutUint16(ack[0:2], transactionID)
	ack[2] = 0x00                                // Message Encoding: Binary Data
	ack[3] = 0x03                                // Message Type: Acknowledge
	binary.BigEndian.PutUint16(ack[4:6], 0x0000) // Status Code: Success

	_, err := writer.Write(ack)
	return err
}

func (t *IntelliTracAProtocol) handlePositionalData(data []byte, modemID string, dataLen uint16, messageID uint16, transactionID uint16, writer io.Writer, store store.Store) error {
	position := &PositionRecord{
		TransactionID: transactionID,
		ModemID:       modemID,
		MessageID:     messageID,
		DataLength:    dataLen,
	}

	// GPS Date/Time (bytes 0-5)
	position.GPS.Timestamp = parseDateTime(
		data[3], // Year
		data[4], // Month
		data[5], // Day
		data[0], // Hour
		data[1], // Minute
		data[2], // Second
	)

	// GPS Position (bytes 6-13)
	position.GPS.Latitude = float64(int32(binary.BigEndian.Uint32(data[6:10]))) / 100000.0
	position.GPS.Longitude = float64(int32(binary.BigEndian.Uint32(data[10:14]))) / 100000.0

	// Altitude (bytes 14-16, 3 bytes signed)
	altBytes := []byte{0}
	if data[14]&0x80 != 0 { // negative number
		altBytes[0] = 0xFF
	}
	altBytes = append(altBytes, data[14:17]...)
	position.GPS.Altitude = int32(binary.BigEndian.Uint32(altBytes))

	// Speed and Direction (bytes 17-20)
	position.GPS.Speed = (float32(binary.BigEndian.Uint16(data[17:19])) / 10.0) * 3.8 // 0.1 m/s units
	position.GPS.Direction = float32(binary.BigEndian.Uint16(data[19:21])) / 10.0     // 0.1 degree units

	// Odometer (bytes 21-24)
	position.Odometer = binary.BigEndian.Uint32(data[21:25]) / 1000

	// HDOP and Satellites (bytes 25-26)
	position.HDOP = data[25] / 10.0
	position.Satellites = data[26]

	position.IOStatus = binary.BigEndian.Uint16(data[27:29])

	// Vehicle Status (byte 29)
	position.VehicleStatus = data[29]

	position.AnalogInput1 = (float32(binary.BigEndian.Uint16(data[30:32])))
	position.AnalogInput2 = (float32(binary.BigEndian.Uint16(data[32:34])))

	// RTC Date/Time (bytes 34-39)
	position.RTC = parseDateTime(
		data[37], // Year
		data[38], // Month
		data[39], // Day
		data[34], // Hour
		data[35], // Minute
		data[36], // Second
	)

	// Position Sending Time (bytes 40-45)
	position.PositionSending = parseDateTime(
		data[43], // Year
		data[44], // Month
		data[45], // Day
		data[40], // Hour
		data[41], // Minute
		data[42], // Second
	)

	position.RawData = fmt.Sprintf("%v", data)

	// after you’ve populated `position`…
	logger.Sugar().Infow("position",
		// envelope
		"transactionID", position.TransactionID,
		"modemID", position.ModemID,
		"messageID", fmt.Sprintf("0x%04X", position.MessageID),
		"dataLength", position.DataLength,

		// gps
		"gpsTimestamp", position.GPS.Timestamp.Format(time.RFC3339),
		"latitude", position.GPS.Latitude,
		"longitude", position.GPS.Longitude,
		"altitude", position.GPS.Altitude,
		"speed_kmh", position.GPS.Speed,
		"direction_deg", position.GPS.Direction,

		// odometer + fix quality
		"odometer_km", position.Odometer,
		"hdop", float32(position.HDOP)/10.0,
		"satellites", position.Satellites,

		// digital I/O
		"ioStatus", fmt.Sprintf("0x%04X", position.IOStatus),
		"vehicleStatus", fmt.Sprintf("0x%02X", position.VehicleStatus),

		//input
		"AnalogInput1", position.AnalogInput1,
		"AnalogInput2", position.AnalogInput2,

		// device times
		"rtc", position.RTC.Format(time.RFC3339),
		"sentAt", position.PositionSending.Format(time.RFC3339),
	)

	// Convert to DeviceStatus and send to store
	status := position.ToDeviceStatus(t.Imei)
	store.GetProcessChan() <- status

	logger.Sugar().Infof("46-byte position parsed - Lat: %.5f, Lon: %.5f",
		position.GPS.Latitude, position.GPS.Longitude)

	// Send acknowledgment
	ack := make([]byte, BinaryAckSize)
	binary.BigEndian.PutUint16(ack[0:2], transactionID)
	ack[2] = 0x00                                // Message Encoding: Binary Data
	ack[3] = 0x03                                // Message Type: Acknowledge
	binary.BigEndian.PutUint16(ack[4:6], 0x0000) // Status Code: Success

	_, err := writer.Write(ack)
	return err
}

// func (p *PositionRecord) parseEventData() {
// 	p.AdditionalMetrics = make(map[uint16]interface{})
// 	switch p.MessageID {
// 	case 0x96: // Tow event
// 		if len(p.EventData) >= 3 {
// 			p.AdditionalMetrics[0x01] = binary.BigEndian.Uint16(p.EventData[0:2]) // Distance
// 		}
// 	case 0xC7: // Impact event
// 		if len(p.EventData) >= 3 {
// 			p.AdditionalMetrics[0x01] = int8(p.EventData[0]) // X-G Force
// 			p.AdditionalMetrics[0x02] = int8(p.EventData[1]) // Y-G Force
// 			p.AdditionalMetrics[0x03] = int8(p.EventData[2]) // Z-G Force
// 		}
// 	case 0x99: // Idle event
// 		if len(p.EventData) >= 2 {
// 			p.AdditionalMetrics[0x01] = binary.BigEndian.Uint16(p.EventData[0:2]) // Duration
// 		}
// 	}
// }

func (t *IntelliTracAProtocol) consumeASCIIStream(reader *bufio.Reader, writer io.Writer, store store.Store) error {
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	line = strings.TrimSpace(line)
	parts := strings.Split(line, ",")

	logger.Sugar().Infoln("full ascii", line)

	// Position message format (page 13)
	if len(parts) >= 15 {
		// 	position := PositionRecord{
		// 		ModemID: parseUint64(parts[0]),
		// 		GPS: GPSData{
		// 			Timestamp: parseTime(parts[1]),
		// 			Longitude: parseFloat(parts[2]),
		// 			Latitude:  parseFloat(parts[3]),
		// 			Speed:     float32(parseFloat(parts[4])),
		// 			Direction: float32(parseFloat(parts[5])),
		// 			Altitude:  int32(parseFloat(parts[6])),
		// 		},
		// 		Satellites: uint8(parseInt(parts[7])),
		// 		MessageID:  uint16(parseInt(parts[8])),
		// 		IOStatus:   uint16(parseInt(parts[9])),
		// 	}

		// Send to store
		status := &types.DeviceStatus{
			Imei: t.Imei}
		// status := position.ToDeviceStatus(t.Imei)
		store.GetProcessChan() <- status

		// Send binary ack (transaction ID 0 for ASCII)
		ack := make([]byte, BinaryAckSize)
		binary.BigEndian.PutUint16(ack[0:2], 0)
		ack[2] = MsgEncodingBinaryPos
		ack[3] = MsgTypeAck
		binary.BigEndian.PutUint16(ack[4:6], 0)
		_, err := writer.Write(ack)
		return err
	}

	return nil
}

func (t *IntelliTracAProtocol) isImeiAuthorized(imei string) bool {
	// Implement your authorization logic here
	return true
}

func (t *IntelliTracAProtocol) SendCommandToDevice(writer io.Writer, command string) error {
	if t.IsBinary {
		return t.sendBinaryCommand(writer, command)
	}
	return t.sendASCIICommand(writer, command)
}

func (t *IntelliTracAProtocol) sendBinaryCommand(writer io.Writer, command string) error {
	// Generate random transaction ID
	transactionID := uint16(time.Now().UnixNano() % 65536)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, transactionID)
	buf.WriteByte(MsgEncodingATCommand)
	buf.WriteByte(MsgTypeRequest)

	// Write data length (2 bytes)
	data := []byte(command)
	dataLen := uint16(len(data))
	binary.Write(buf, binary.BigEndian, dataLen)

	// Write command
	buf.Write(data)

	_, err := writer.Write(buf.Bytes())
	return err
}

func (t *IntelliTracAProtocol) handleTextMessage(reader *bufio.Reader, writer io.Writer, store store.Store, transactionID uint16) error {
	// Read data length (2 bytes)
	dataLenBytes := make([]byte, 2)
	if _, err := io.ReadFull(reader, dataLenBytes); err != nil {
		return err
	}
	dataLen := binary.BigEndian.Uint16(dataLenBytes)

	// Read modem ID (8 bytes)
	modemIDBytes := make([]byte, 8)
	if _, err := io.ReadFull(reader, modemIDBytes); err != nil {
		return err
	}
	// modemID := binary.BigEndian.Uint64(modemIDBytes)

	// Read text data
	textData := make([]byte, dataLen)
	if _, err := io.ReadFull(reader, textData); err != nil {
		return err
	}

	// Read timestamp data (12 bytes)
	timestampData := make([]byte, 12)
	if _, err := io.ReadFull(reader, timestampData); err != nil {
		return err
	}

	// Log the text message
	logger.Sugar().Infof("Received text message from device %s: %s", t.Imei, string(textData))

	// Send acknowledgment
	ack := make([]byte, BinaryAckSize)
	binary.BigEndian.PutUint16(ack[0:2], transactionID)
	ack[2] = MsgEncodingText
	ack[3] = MsgTypeAck
	binary.BigEndian.PutUint16(ack[4:6], 0) // Success
	_, err := writer.Write(ack)
	return err
}

func (t *IntelliTracAProtocol) handleATResponse(reader *bufio.Reader, writer io.Writer, store store.Store, transactionID uint16) error {
	// Read data length (2 bytes)
	dataLenBytes := make([]byte, 2)
	if _, err := io.ReadFull(reader, dataLenBytes); err != nil {
		return err
	}
	dataLen := binary.BigEndian.Uint16(dataLenBytes)

	// Read response data
	responseData := make([]byte, dataLen)
	if _, err := io.ReadFull(reader, responseData); err != nil {
		return err
	}

	// Handle different types of AT responses
	responseStr := string(responseData)
	switch {
	case strings.HasPrefix(responseStr, "OK"):
		logger.Sugar().Debugf("Device %s acknowledged command", t.Imei)
	case strings.HasPrefix(responseStr, "ERROR"):
		logger.Sugar().Warnf("Device %s returned error: %s", t.Imei, responseStr)
	default:
		logger.Sugar().Infof("Received AT response from device %s: %s", t.Imei, responseStr)
	}

	// Send to response channel if needed
	store.GetResponseChan() <- &types.DeviceResponse{
		Imei:     t.Imei,
		Response: responseStr,
	}

	return nil
}

func padIMEI(imei string) string {
	if len(imei) < 15 {
		return strings.Repeat("0", 15-len(imei)) + imei
	}
	return imei
}

func (t *IntelliTracAProtocol) sendASCIICommand(writer io.Writer, command string) error {
	_, err := writer.Write([]byte(command + "\r\n"))
	return err
}
