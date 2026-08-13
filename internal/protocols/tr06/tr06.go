package tr06

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/404minds/avl-receiver/internal/store"
	"go.uber.org/zap"

	"github.com/404minds/avl-receiver/internal/crc"
	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/types"
	"github.com/pkg/errors"
)

var logger = configuredLogger.Logger

// deviceTimeZone is the timezone the terminal's Date Time field is expressed in.
//
// TR06 §5.2.1.4 defines only the encoding of Date Time (year+2000, month, day, hour,
// minute, second) and never states which timezone those digits are in, and the TR06
// login packet (§5.1.1, 18 bytes) carries no timezone field - so there is nothing in
// the protocol to derive it from. It is a per-fleet terminal setting.
//
// ponytail: default stays UTC, which is the previous behaviour. Set
// TR06_DEVICE_TZ_OFFSET (e.g. "+05:30") when the terminals are configured to report
// local time.
var deviceTimeZone = loadDeviceTimeZone(os.Getenv("TR06_DEVICE_TZ_OFFSET"))

func loadDeviceTimeZone(offset string) *time.Location {
	if offset == "" {
		return time.UTC
	}
	t, err := time.Parse("-07:00", offset)
	if err != nil {
		logger.Sugar().Errorf("TR06_DEVICE_TZ_OFFSET %q is not a ±HH:MM offset, falling back to UTC: %v", offset, err)
		return time.UTC
	}
	_, seconds := t.Zone()
	logger.Sugar().Infof("TR06 device time zone offset applied: %s (%d seconds)", offset, seconds)
	return time.FixedZone(offset, seconds)
}

type TR06Protocol struct {
	LoginInformation *LoginData
	DeviceType       types.DeviceType
}

func (p *TR06Protocol) GetDeviceID() string {
	if p.LoginInformation == nil {
		logger.Error("LoginInformation is nil in GetDeviceID")
		return ""
	}

	if p.LoginInformation.TerminalID == "" {
		logger.Error("Login Information does not have TerminalID in GetDeviceID")
	}

	return p.LoginInformation.TerminalID
}

func (p *TR06Protocol) GetDeviceType() types.DeviceType {
	return p.DeviceType
}

func (p *TR06Protocol) SetDeviceType(t types.DeviceType) {
	p.DeviceType = t
}

func (p *TR06Protocol) GetProtocolType() types.DeviceProtocolType {
	return types.DeviceProtocolType_GT06
}

func (p *TR06Protocol) Login(reader *bufio.Reader) (ack []byte, byteToSkip int, e error) {
	if !p.IsValidHeader(reader) {
		return nil, 0, errs.ErrUnknownProtocol
	}

	data, _ := reader.Peek(reader.Buffered())
	logger.Sugar().Info("Available data before reading IMEI: ", data)

	// This should have been a GT06 device
	packet, err := p.parsePacket(reader)
	if err != nil {
		logger.Error("failed to parse GT06 packet", zap.Error(err))
		return nil, 0, err
	}

	if packet.MessageType == MSG_LoginData {
		if packet.Information == nil {
			logger.Error("packet information is nil", zap.Error(errs.ErrGT06InvalidLoginInfo))
			return nil, 0, errs.ErrGT06InvalidLoginInfo
		}

		loginData, ok := packet.Information.(*LoginData)
		if !ok {
			logger.Error("packet information is not of type *LoginData", zap.Error(errs.ErrGT06InvalidLoginInfo))
			return nil, 0, errs.ErrGT06InvalidLoginInfo
		}

		if loginData == nil {
			logger.Error("loginData is nil", zap.Error(errs.ErrGT06InvalidLoginInfo))
			return nil, 0, errs.ErrGT06InvalidLoginInfo
		}
		logger.Sugar().Info("from Login LoginData: ", loginData)
		p.LoginInformation = loginData

		byteBuffer := bytes.NewBuffer([]byte{})
		err = p.sendResponse(packet, byteBuffer)
		if err != nil {
			logger.Error("failed to send response for GT06 packet", zap.Error(err))
			return nil, 0, err
		}

		return byteBuffer.Bytes(), 0, nil // nothing to skip since the stream is already consumed
	} else {
		logger.Error("packet message type is not MSG_LoginData", zap.Error(errs.ErrGT06InvalidLoginInfo))
		return nil, 0, errs.ErrGT06InvalidLoginInfo
	}
}

func (p *TR06Protocol) ConsumeStream(reader *bufio.Reader, writer io.Writer, dataStore store.Store) error {
	for {

		packet, err := p.parsePacket(reader)
		if err != nil {
			logger.Sugar().Info("Consume Stream :", err)
			return err
		}
		// TR06 §5.4.2 the server responds to the status/heartbeat packet, §5.3.2 the server
		// responds to the alarm (GPS+LBS+status) packet. §5.2 defines no response for the
		// plain location packet, so 0x12 is deliberately not acknowledged.
		if packet.MessageType == MSG_HeartbeatData || packet.MessageType == MSG_GPS_LBS_StatusData {
			err = p.sendResponse(packet, writer)
			if err != nil {
				logger.Sugar().Info("error while sending response", err)
				return err
			}
		}

		if packet.Information == nil {
			continue // unsupported protocol number, already logged and skipped
		}

		asyncStore := dataStore.GetProcessChan()
		protoPacket := packet.ToProtobufDeviceStatus(p.GetDeviceID(), p.DeviceType)
		asyncStore <- protoPacket
	}
}

func (p *TR06Protocol) sendResponse(parsedPacket *Packet, writer io.Writer) error {
	defer func() {
		if condition := recover(); condition != nil {
			err := fmt.Errorf("panic: %v", condition)
			logger.Error("Failed to write response packet", zap.Error(err))
		}
	}()

	responsePacket := ResponsePacket{
		StartBit:                0x7878,
		PacketLength:            0x05,
		ProtocolNumber:          int8(parsedPacket.MessageType),
		InformationSerialNumber: parsedPacket.InformationSerialNumber,
		StopBits:                0xd0a,
	}

	responsePacket.Crc = crc.CrcWanway(responsePacket.ToBytes()[2:6])

	logger.Sugar().Info("Sending response packet: ", responsePacket.ToBytes())
	_, err := writer.Write(responsePacket.ToBytes())
	if err != nil {
		return errors.Wrapf(err, "failed to write response packet")
	}
	return nil
}

func (p *TR06Protocol) parsePacket(reader *bufio.Reader) (packet *Packet, err error) {
	defer func() {
		if r := recover(); r != nil {
			if rErr, ok := r.(error); ok {
				err = rErr
			} else {
				err = fmt.Errorf("parse packet unknown panic: %v", r)
			}
			if err != io.EOF {
				err = errors.Wrapf(errs.ErrGT06BadDataPacket, "from parsePAcket")
				logger.Sugar().Info("parse packet 0 ", err)
			}
			logger.Sugar().Errorf("parse packet Recovered from panic: %v", err)
		}
	}()

	packet = &Packet{}

	// Start bit
	err = binary.Read(reader, binary.BigEndian, &packet.StartBit)
	logger.Sugar().Infof("parse packet Start bit: %x", packet.StartBit)
	if err != nil {
		logger.Sugar().Errorf("parse packet Failed to read start bit: %v", err)
		return nil, err
	}

	// Packet length. TR06 §4.1 fixes the start bit at 0x7878 and §4.2 makes the length a
	// single byte; there is no 0x7979 two-byte-length frame in TR06.
	if packet.StartBit != 0x7878 {
		return nil, errors.Wrapf(errs.ErrGT06BadDataPacket, "from parsePacket Invalid StartBit packet.StartBit: %x", packet.StartBit)
	}
	var packetLength byte
	err = binary.Read(reader, binary.BigEndian, &packetLength)
	if err != nil {
		logger.Sugar().Errorf("parse packet Failed to read packet length: %v", err)
		return nil, err
	}
	packet.PacketLength = packetLength
	logger.Sugar().Infof("parse packet Packet length: %d", packet.PacketLength)
	if packet.PacketLength < 5 {
		return nil, errors.Wrapf(errs.ErrGT06BadDataPacket, "from parsePacket packet length %d below the 5 byte minimum", packet.PacketLength)
	}

	// Packet data
	packetData := make([]byte, packet.PacketLength-4) // 2 for CRC, 2 for serial number
	logger.Sugar().Infof("parse packet packet data after removing 2 for CRC, 2 for serial number: %x", packetData)

	_, err = io.ReadFull(reader, packetData)
	if err != nil {
		logger.Sugar().Errorf("parse packet Failed to read packet data: %v", err)
		return nil, err
	}
	logger.Sugar().Infof("parse packet Packet data: %x", packetData)

	// Packet data to packet
	logger.Sugar().Info("Parse packet: ", packetData)
	err = p.parsePacketData(bufio.NewReader(bytes.NewReader(packetData)), packet)
	if err != nil {
		logger.Sugar().Errorf("parse packet Failed to parse packet data: %v", err)
		return nil, err
	}

	// Information serial number
	err = binary.Read(reader, binary.BigEndian, &packet.InformationSerialNumber)
	if err != nil {
		logger.Sugar().Errorf("parse packet Failed to read information serial number: %v", err)
		return nil, err
	}
	logger.Sugar().Infof("parse packet Information serial number: %x", packet.InformationSerialNumber)

	// CRC
	err = binary.Read(reader, binary.BigEndian, &packet.Crc)
	if err != nil {
		logger.Sugar().Errorf("parse packet Failed to read CRC: %v", err)
		return nil, err
	}
	logger.Sugar().Infof("parse packet CRC: %x", packet.Crc)

	// Stop bits
	err = binary.Read(reader, binary.BigEndian, &packet.StopBits)
	if err != nil {
		logger.Sugar().Errorf("parse packet Failed to read stop bits: %v", err)
		return nil, err
	}
	logger.Sugar().Infof("parse packet Stop bits: %x", packet.StopBits)

	if packet.StopBits != 0x0d0a {
		err = errors.Wrapf(errs.ErrGT06BadDataPacket, "from parsePacket 3")
		logger.Sugar().Errorf("parse packet Invalid stop bits: %x  parse packet 1 ERRTRO6 %v", packet.StopBits, err)
		return nil, err
	}

	//Validate CRC
	expectedCrc := crc.CrcWanway(
		slices.Concat(
			[]byte{byte(packet.PacketLength)},
			packetData,
			[]byte{
				byte(packet.InformationSerialNumber >> 8),
				byte(packet.InformationSerialNumber & 0xff),
			},
		),
	)
	if expectedCrc != packet.Crc {
		logger.Sugar().Errorf("parse packet Invalid CRC. Expected %x, got %x", expectedCrc, packet.Crc)
		return nil, errs.ErrBadCrc
	}

	return packet, nil
}

func (p *TR06Protocol) parsePacketData(reader *bufio.Reader, packet *Packet) error {

	protocolNumByte, err := reader.ReadByte()
	if err != nil {
		logger.Sugar().Errorf("parsePacketData failed to read protocol number: %v", err)
		return err
	}
	logger.Sugar().Info("parsePacketData protocol number byte: ", protocolNumByte)

	msgType := MessageType(protocolNumByte)
	logger.Sugar().Info("message type ", msgType)

	packet.MessageType = msgType

	// TODO: parse packetInfoBytes
	packet.Information, err = p.parsePacketInformation(reader, packet.MessageType)
	if err != nil {
		return err
	}

	return nil
}

func (p *TR06Protocol) consumePacket(reader *bufio.Reader) ([]byte, error) {
	data := make([]byte, 0)
	term := []byte{0x0d, 0x0a}

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		data = append(data, b)
		if bytes.HasSuffix(data, term) {
			break
		}
	}
	return data, nil
}

func (p *TR06Protocol) parsePacketInformation(reader *bufio.Reader, messageType MessageType) (interface{}, error) {
	if messageType == MSG_LoginData {
		parsedInfo, err := p.parseLoginInformation(reader)
		return parsedInfo, err
	} else if messageType == MSG_PositioningData {
		parsedInfo, err := p.parsePositioningData(reader)
		return parsedInfo, err
	} else if messageType == MSG_AlarmData {
		parsedInfo, err := p.parseAlarmData(reader)
		return parsedInfo, err
	} else if messageType == MSG_HeartbeatData {
		parsedInfo, err := p.parseHeartbeatData(reader)
		logger.Sugar().Info("parsePacketInformation error: ", err)
		return parsedInfo, err
	} else if messageType == MSG_EG_HeartbeatData {
		parsedInfo, err := p.parseHeartbeatData(reader)
		return parsedInfo, err
	} else if messageType == MSG_GPS_LBS_StatusData {
		parsedInfo, err := p.parseAlarmData(reader)
		return parsedInfo, err
	} else if messageType == MSG_TransmissionInstruction {
		parsedInfo, err := p.parseInformationTransmissionPacket(reader)
		return parsedInfo, err
	} else {
		// ponytail: the frame is already fully read by packet length, so an unsupported
		// protocol number is skipped rather than failing the packet. TR06 §iii.7 warns
		// that dropping the connection makes the terminal reconnect in a loop, and §4.6
		// prescribes discarding a bad packet - not the link.
		logger.Sugar().Warnf("parsePacketInformation: unsupported protocol number %#x, packet skipped", byte(messageType))
		return nil, nil
	}
}

func (p *TR06Protocol) parseLoginInformation(reader *bufio.Reader) (interface{}, error) {
	var loginInfo LoginData

	var imeiBytes [8]byte
	err := binary.Read(reader, binary.BigEndian, &imeiBytes)
	if err != nil {
		logger.Error("failed to read IMEI bytes", zap.Error(err))
		return nil, errs.ErrGT06InvalidLoginInfo
	}
	logger.Sugar().Info("parseLoginInformation imeiBytes: ", imeiBytes[:])
	loginInfo.TerminalID = hex.EncodeToString(imeiBytes[:])[1:] // IMEI is 15 chars
	logger.Sugar().Info("parseLoginInformation loginInfo.TerminalID: ", loginInfo.TerminalID)
	logger.Sugar().Info("parseLoginInformation loginInfo: ", loginInfo)
	return &loginInfo, nil
}
func (p *TR06Protocol) parsePositioningData(reader *bufio.Reader) (positionInfo interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if err != io.EOF {
				logger.Sugar().Info("from parsePositioningData err: ", err)
				err = errors.Wrapf(errs.ErrGT06BadDataPacket, "from parsePositioningData")
			}
		}
	}()

	var parsed PositioningInformation

	// Date Time
	timestamp, err := p.parseTimestamp(reader)
	if err != nil {
		logger.Sugar().Errorf("parsePositioningData failed to parse timestamp: %v", err)
		return nil, errors.Wrap(err, "failed to parse timestamp")
	}
	parsed.GpsInformation.Timestamp = timestamp
	// Quantity of GPS information and number of satellites
	var gpsInfo byte
	checkErr(binary.Read(reader, binary.BigEndian, &gpsInfo))
	logger.Sugar().Infof("parsePositioningData GPS info: %x", gpsInfo)
	parsed.GpsInformation.GPSInfoLength = gpsInfo >> 4
	parsed.GpsInformation.NumberOfSatellites = gpsInfo & 0x0F

	// Latitude
	var latitude uint32
	checkErr(binary.Read(reader, binary.BigEndian, &latitude))
	parsed.GpsInformation.Latitude = float32(latitude) / 30000 / 60
	logger.Sugar().Infof("parsePositioningData Latitude: %x", latitude)

	// Longitude
	var longitude uint32
	checkErr(binary.Read(reader, binary.BigEndian, &longitude))
	parsed.GpsInformation.Longitude = float32(longitude) / 30000 / 60
	logger.Sugar().Infof("parsePositioningData Longitude: %x", longitude)

	// Speed
	checkErr(binary.Read(reader, binary.BigEndian, &parsed.GpsInformation.Speed))
	logger.Sugar().Infof("parsePositioningData Speed: %x", parsed.GpsInformation.Speed)

	// Course and Status
	var courseAndStatus [2]byte
	checkErr(binary.Read(reader, binary.BigEndian, &courseAndStatus))
	logger.Sugar().Infof("parsePositioningData Course and Status: %x", courseAndStatus)
	parsed.GpsInformation.Course = parseCourseAndStatus(courseAndStatus)
	applyHemisphere(&parsed.GpsInformation)

	// MCC
	checkErr(binary.Read(reader, binary.BigEndian, &parsed.LBSInfo.MCC))
	logger.Sugar().Infof("parsePositioningData MCC: %x", parsed.LBSInfo.MCC)

	// MNC
	checkErr(binary.Read(reader, binary.BigEndian, &parsed.LBSInfo.MNC))
	logger.Sugar().Infof("parsePositioningData MNC: %x", parsed.LBSInfo.MNC)

	// LAC
	checkErr(binary.Read(reader, binary.BigEndian, &parsed.LBSInfo.LAC))
	logger.Sugar().Infof("parsePositioningData LAC: %x", parsed.LBSInfo.LAC)

	// Cell ID
	checkErr(binary.Read(reader, binary.BigEndian, &parsed.LBSInfo.CellID))
	logger.Sugar().Infof("parsePositioningData Cell ID: %x", parsed.LBSInfo.CellID)

	return &parsed, nil
}

// parseCourseAndStatus decodes the 2 byte "Course, Status" field of TR06 §5.2.1.9.
// BYTE_1: bit7/bit6 are 0, bit5 real-time(0)/differential(1) GPS, bit4 GPS positioned,
// bit3 East(0)/West(1) longitude, bit2 South(0)/North(1) latitude, bit1..bit0 are the
// two high bits of the course. BYTE_2 holds the low 8 bits of the course.
func parseCourseAndStatus(courseAndStatus [2]byte) GPSCourse {
	var course GPSCourse
	course.IsRealtime = (courseAndStatus[0] & 0x20) == 0
	course.IsDifferential = (courseAndStatus[0] & 0x20) != 0
	course.Positioned = (courseAndStatus[0] & 0x10) != 0
	course.Longitude = (courseAndStatus[0] & 0x08) != 0 // 0: East, 1: West
	course.Latitude = (courseAndStatus[0] & 0x04) != 0  // 0: South, 1: North
	course.Degree = uint16(courseAndStatus[1]) | (uint16(courseAndStatus[0]&0x03) << 8)
	return course
}

// applyHemisphere signs the coordinates from the status bits. TR06 §5.2.1.6/§5.2.1.7
// transmit latitude and longitude as unsigned magnitudes (0-162000000 for 0-90° and
// 0-324000000 for 0-180°); the hemisphere lives in the Course/Status bits.
func applyHemisphere(gps *GPSInformation) {
	if !gps.Course.Latitude { // south latitude
		gps.Latitude = -gps.Latitude
	}
	if gps.Course.Longitude { // west longitude
		gps.Longitude = -gps.Longitude
	}
}

// parseAlarmData decodes the TR06 §5.3.1 alarm packet (0x16), the combined information
// packet of GPS, LBS and status:
//
//	Date Time 6 | GPS len/satellites 1 | Latitude 4 | Longitude 4 | Speed 1 |
//	Course,Status 2 | LBS Length 1 | MCC 2 | MNC 1 | LAC 2 | Cell ID 3 |
//	Terminal Information 1 | Voltage Level 1 | GSM Signal Strength 1 | Alarm/Language 2
func (p *TR06Protocol) parseAlarmData(reader *bufio.Reader) (alarmInfo *AlarmInformation, err error) {
	defer func() {
		if r := recover(); r != nil {
			alarmInfo = nil
			err = r.(error)
			if err != io.EOF {
				logger.Sugar().Info("error from parseAlarmData err: ", err)
				err = errors.Wrapf(errs.ErrGT06BadDataPacket, "from parseAlarmData")
			}
		}
	}()

	var parsed AlarmInformation

	parsed.GpsInformation, err = p.parseGPSInformation(reader)
	checkErr(err)

	// LBS Length: §5.3.1 adds this byte to the LBS block and it counts itself, so its
	// value is 1 + MCC(2) + MNC(1) + LAC(2) + Cell ID(3) = 0x09 for this layout.
	lbsLength, err := reader.ReadByte()
	checkErr(err)
	logger.Sugar().Infof("parseAlarmData LBS length: %#x", lbsLength)

	parsed.LBSInformation, err = p.parseLBSInformation(reader)
	checkErr(err)

	parsed.StatusInformation, err = p.parseStatusInformation(reader)
	checkErr(err)

	return &parsed, nil
}

// parseHeartbeatData decodes the TR06 §5.4.1 status information packet (0x13):
// Terminal Information 1 | Voltage Level 1 | GSM Signal Strength 1 | Alarm/Language 2
func (p *TR06Protocol) parseHeartbeatData(reader *bufio.Reader) (heartbeat *HeartbeatData, err error) {
	defer func() {
		if r := recover(); r != nil {
			heartbeat = nil
			err = r.(error)
			if err != io.EOF {
				logger.Sugar().Info("error from parseHeartbeatData 1 err: ", err)
				err = errors.Wrapf(errs.ErrGT06BadDataPacket, "from parseHeartbeatData")
			}
		}
	}()

	var parsed HeartbeatData

	var terminalInfoByte byte
	if err := binary.Read(reader, binary.BigEndian, &terminalInfoByte); err != nil {
		return nil, err
	}
	logger.Sugar().Infof("parseHeartbeatData Terminal Info Byte: %x", terminalInfoByte)
	parsed.TerminalInformation, err = p.parseTerminalInfoFromByte(terminalInfoByte)
	if err != nil {
		return nil, err
	}

	var batteryLevelByte byte
	if err := binary.Read(reader, binary.BigEndian, &batteryLevelByte); err != nil {
		return nil, err
	}
	logger.Sugar().Infof("parseHeartbeatData  Battery Level Byte: %x", batteryLevelByte)
	parsed.BatteryLevel = BatteryLevel(batteryLevelByte)
	if parsed.BatteryLevel == VL_Invalid {
		return nil, errs.ErrGT06InvalidVoltageLevel
	}

	var gsmSignalStrengthByte byte
	if err := binary.Read(reader, binary.BigEndian, &gsmSignalStrengthByte); err != nil {
		return nil, err
	}
	logger.Sugar().Infof("parseHeartbeatData GSM Signal Strength Byte: %x", gsmSignalStrengthByte)
	parsed.GSMSignalStrength = GSMSignalStrength(gsmSignalStrengthByte)
	if parsed.GSMSignalStrength == GSM_Invalid {
		return nil, errs.ErrGT06InvalidGSMSignalStrength
	}

	if err := binary.Read(reader, binary.BigEndian, &parsed.AlarmLanguage); err != nil {
		return nil, err
	}
	logger.Sugar().Infof("parseHeartbeatData Alarm/Language: %x", parsed.AlarmLanguage)

	if _, err := reader.Peek(1); err != io.EOF {
		logger.Sugar().Errorf("parseHeartbeatData Extra bytes detected in packet")
		logger.Sugar().Info("error from parseHeartbeatData 2")
		return nil, errors.Wrapf(errs.ErrGT06BadDataPacket, "from parseHeartbeatData 2")
	}

	return &parsed, nil
}

func (p *TR06Protocol) parseInformationTransmissionPacket(reader *bufio.Reader) (packet InformationTransmissionPacket, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if err != io.EOF {
				logger.Sugar().Info("error from parseInformationTransmissionPacket: ", err)
				err = errors.New("GT06 Bad Data Packet")
			}
		}
	}()

	var informationType byte
	if err := binary.Read(reader, binary.BigEndian, &informationType); err != nil {
		logger.Sugar().Info("parseInformationTransmissionPacket: Failed to read information type", err)
		return packet, err
	}

	packet.InformationContent.InformationType = InformationType(informationType)
	logger.Sugar().Info("parseInformationTransmissionPacket: ", packet.InformationContent.InformationType)

	dataContent := make([]byte, 2)
	logger.Sugar().Info("parseInformationTransmissionPacket: Reading data content: ", dataContent)
	if _, err := io.ReadFull(reader, dataContent); err != nil {
		logger.Sugar().Info("parseInformationTransmissionPacket: Failed to read data content ", err)
		return packet, err
	}

	logger.Sugar().Info("parseInformationTransmissionPacket: Parsing data content based on information type ", informationType)
	switch InformationType(informationType) {
	case ExternalPowerVoltage:
		if len(dataContent) < 2 {
			logger.Sugar().Info("parseInformationTransmissionPacket: Insufficient data for ExternalPowerVoltage")
			return packet, errors.New("Insufficient data for ExternalPowerVoltage")
		}
		voltage := binary.BigEndian.Uint16(dataContent)
		logger.Sugar().Info("voltage: ", voltage)
		packet.InformationContent.DataContent = (voltage) / 100
	case TerminalStatusSync:
		status := packet.InformationContent.DataContent
		packet.InformationContent.DataContent = status
	case DoorStatus:
		if len(dataContent) < 1 {
			logger.Sugar().Info("parseInformationTransmissionPacket: Insufficient data for DoorStatus")
			return packet, errors.New("Insufficient data for DoorStatus")
		}
		doorStatus := packet.InformationContent.DataContent
		packet.InformationContent.DataContent = doorStatus
	default:
		break
	}

	if remain, err := reader.Peek(1); err != io.EOF {
		logger.Sugar().Info("parseInformationTransmissionPacket remaining bytes: ", remain)
		logger.Sugar().Info("parseInformationTransmissionPacket: Extra bytes detected in packet")
		return packet, errors.New("GT06 Bad Data Packet")
	}

	logger.Sugar().Info("parseInformationTransmissionPacket: Successfully parsed packet")
	return packet, nil
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func (p *TR06Protocol) parseGPSInformation(reader *bufio.Reader) (gpsInfo GPSInformation, err error) {
	timestamp, err := p.parseTimestamp(reader)
	checkErr(err)
	gpsInfo.Timestamp = timestamp

	x, err := reader.ReadByte()
	checkErr(err)
	gpsInfo.GPSInfoLength = x >> 4
	gpsInfo.NumberOfSatellites = x & 0x0f

	var u32 uint32
	// latitude
	checkErr(binary.Read(reader, binary.BigEndian, &u32))
	gpsInfo.Latitude = float32(u32) / 1800000

	// longitude
	checkErr(binary.Read(reader, binary.BigEndian, &u32))
	gpsInfo.Longitude = float32(u32) / 1800000

	// speed
	checkErr(binary.Read(reader, binary.BigEndian, &gpsInfo.Speed))
	logger.Sugar().Info("speed from parseGPSInformation: ", gpsInfo.Speed)

	// course/status
	var courseAndStatus [2]byte
	checkErr(binary.Read(reader, binary.BigEndian, &courseAndStatus))
	gpsInfo.Course = parseCourseAndStatus(courseAndStatus)
	applyHemisphere(&gpsInfo)

	return
}

func (p *TR06Protocol) parseTimestamp(reader *bufio.Reader) (timestamp time.Time, err error) {
	year, err := reader.ReadByte()
	checkErr(err)

	yearInt := int(year) + 2000

	month, err := reader.ReadByte()
	checkErr(err)

	day, err := reader.ReadByte()
	checkErr(err)

	hour, err := reader.ReadByte()
	checkErr(err)

	minute, err := reader.ReadByte()
	checkErr(err)

	second, err := reader.ReadByte()
	checkErr(err)

	timestamp = time.Date(yearInt, time.Month(month), int(day), int(hour), int(minute), int(second), 0, deviceTimeZone)
	logger.Sugar().Info("timestamp: ", timestamp)
	return timestamp, nil
}

func (p *TR06Protocol) parseLBSInformation(reader *bufio.Reader) (lbsInfo LBSInformation, err error) {
	// MCC
	checkErr(binary.Read(reader, binary.BigEndian, &lbsInfo.MCC))
	// MNC
	checkErr(binary.Read(reader, binary.BigEndian, &lbsInfo.MNC))
	// LAC
	checkErr(binary.Read(reader, binary.BigEndian, &lbsInfo.LAC))
	// cell id
	checkErr(binary.Read(reader, binary.BigEndian, &lbsInfo.CellID))
	return
}

func (p *TR06Protocol) parseStatusInformation(reader *bufio.Reader) (statusInfo StatusInformation, err error) {
	var b byte

	// terminal information content
	checkErr(binary.Read(reader, binary.BigEndian, &b))
	statusInfo.TerminalInformation, err = p.parseTerminalInfoFromByte(b)
	checkErr(err)

	// voltage level
	checkErr(binary.Read(reader, binary.BigEndian, &b))
	statusInfo.BatteryLevel = BatteryLevel(b)
	if statusInfo.BatteryLevel == VL_Invalid {
		return statusInfo, errs.ErrGT06InvalidAlarmType
	}

	// GSM signal strength
	checkErr(binary.Read(reader, binary.BigEndian, &b))
	statusInfo.GSMSignalStrength = GSMSignalStrength(b)
	if statusInfo.GSMSignalStrength == GSM_Invalid {
		return statusInfo, errs.ErrGT06InvalidGSMSignalStrength
	}

	// Alarm/Language, 2 bytes: former byte is the alarm status, latter byte the language
	// (TR06 §5.3.1.17).
	alarm, err := reader.ReadByte()
	checkErr(err)
	statusInfo.Alarm = AlarmValue(alarm)

	language, err := reader.ReadByte()
	checkErr(err)
	statusInfo.Language = Language(language)
	return
}

// parseTerminalInfoFromByte decodes the Terminal Information byte of TR06 §5.3.1.14 /
// §5.4.1.4: bit7 1 = oil and electricity disconnected / 0 = connected, bit6 1 = GPS
// tracking on, bit5..bit3 alarm (100 SOS, 011 low battery, 010 power cut, 001 shock,
// 000 normal), bit2 1 = charge on, bit1 1 = ACC high, bit0 1 = defense activated.
func (p *TR06Protocol) parseTerminalInfoFromByte(terminalInfoByte byte) (TerminalInformation, error) {
	var terminalInfo TerminalInformation
	terminalInfo.OilElectricityConnected = terminalInfoByte&0x80 == 0x00 // bit 7, 1 means disconnected
	terminalInfo.GPSSignalAvailable = terminalInfoByte&0x40 == 0x40      // bit 6
	terminalInfo.AlarmType = AlarmType((terminalInfoByte >> 3) & 0x07)   // bit 5, 4, 3
	terminalInfo.Charging = terminalInfoByte&0x04 == 0x04                // bit 2
	terminalInfo.ACCHigh = terminalInfoByte&0x02 == 0x02                 // bit 1
	terminalInfo.Armed = terminalInfoByte&0x01 == 0x01                   // bit 0

	if terminalInfo.AlarmType == AL_Invalid {
		return terminalInfo, errs.ErrGT06InvalidAlarmType
	}
	return terminalInfo, nil
}

// IsValidHeader reports whether the stream starts with the TR06 start bit. §4.1 fixes it
// at 0x7878; TR06 has no 0x7979 frame.
func (p *TR06Protocol) IsValidHeader(reader *bufio.Reader) bool {
	header, err := reader.Peek(2)
	if err != nil {
		return false
	}

	return bytes.Equal(header, []byte{0x78, 0x78})
}

// send command to device
func (p *TR06Protocol) SendCommandToDevice(writer io.Writer, command string) error {
	// Command in HEX for "getinfo"
	return nil
}
