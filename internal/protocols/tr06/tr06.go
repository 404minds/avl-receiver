package tr06

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"go.uber.org/zap"
	"io"
	"slices"
	"time"

	"github.com/404minds/avl-receiver/internal/crc"
	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/types"
	"github.com/pkg/errors"
)

var logger = configuredLogger.Logger

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

func (p *TR06Protocol) ConsumeStream(reader *bufio.Reader, writer io.Writer, asyncStore chan types.DeviceStatus) error {
	for {

		packet, err := p.parsePacket(reader)
		if err != nil {
			logger.Sugar().Info("Consume Stream :", err)
			return err
		}
		if packet.MessageType == MSG_HeartbeatData {
			err = p.sendResponse(packet, writer)
			if err != nil {
				logger.Sugar().Info("error while sending response", err)
				return err
			}
		}

		protoPacket := packet.ToProtobufDeviceStatus(p.GetDeviceID(), p.DeviceType)
		asyncStore <- *protoPacket
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

	// Determine packet length based on start bit
	if packet.StartBit == 0x7979 {
		//var packetLength uint16
		//err = binary.Read(reader, binary.BigEndian, &packetLength)
		//if err != nil {
		//	logger.Sugar().Errorf("parse packet Failed to read packet length: %v", err)
		//	return nil, err
		//}
		//packet.PacketLength = (packetLength)
		//logger.Sugar().Infof("parse packet Packet length: %d", packet.PacketLength)

	} else if packet.StartBit == 0x7878 {
		var packetLength byte
		err = binary.Read(reader, binary.BigEndian, &packetLength)
		if err != nil {
			logger.Sugar().Errorf("parse packet Failed to read packet length: %v", err)
			return nil, err
		}
		packet.PacketLength = packetLength
		logger.Sugar().Infof("parse packet Packet length: %d", packet.PacketLength)
	} else {
		return nil, errors.Wrapf(errs.ErrGT06BadDataPacket, "from parsePacket Invalid StartBit packet.StartBit: %d", packet.StartBit) // Invalid start bit
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
	logger.Sugar().Info("parsePacketData protocol number byte: ", protocolNumByte)

	msgType := MessageType(protocolNumByte)
	logger.Sugar().Info("message type ", msgType)

	if msgType == MSG_Invalid {
		logger.Sugar().Errorf("Invalid message type: %x", protocolNumByte)
		remainingData, err := p.consumePacket(reader)
		if err != nil {
			return err
		}
		logger.Sugar().Errorln("Invalid message type: ", hex.Dump(remainingData))
		logger.Sugar().Info("error from parsePacketData ", err)
		return errors.Wrapf(errs.ErrGT06BadDataPacket, "from parsePacketData")
	}

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
	} else if messageType == MSG_TransmissionInstruction {
		parsedInfo, err := p.parseInformationTransmissionPacket(reader)
		return parsedInfo, err
	} else {
		logger.Sugar().Info("error from parsePacketInformation")
		return nil, errors.Wrapf(errs.ErrGT06BadDataPacket, "from parsePAcketInformation")
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

func parseCourseAndStatus(courseAndStatus [2]byte) GPSCourse {
	var course GPSCourse
	course.IsRealtime = (courseAndStatus[0] & 0x40) == 0
	course.IsDifferential = (courseAndStatus[0] & 0x20) != 0
	course.Positioned = (courseAndStatus[0] & 0x10) != 0
	course.Longitude = (courseAndStatus[0] & 0x08) != 0 // 0: East, 1: West
	course.Latitude = (courseAndStatus[0] & 0x04) != 0  // 0: South, 1: North
	course.Degree = uint16(courseAndStatus[1]) | (uint16(courseAndStatus[0]&0x03) << 8)
	return course
}

func (p *TR06Protocol) parseAlarmData(reader *bufio.Reader) (alarmInfo AlarmInformation, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if err != io.EOF {
				logger.Sugar().Info("error from parseAlarmData err: ", err)
				err = errors.Wrapf(errs.ErrGT06BadDataPacket, "from parseAlarmData")
			}
		}
	}()

	alarmInfo.GpsInformation, err = p.parseGPSInformation(reader)
	checkErr(err)

	alarmInfo.LBSInformation, err = p.parseLBSInformation(reader)
	checkErr(err)

	alarmInfo.StatusInformation, err = p.parseStatusInformation(reader)
	checkErr(err)

	return
}

func (p *TR06Protocol) parseHeartbeatData(reader *bufio.Reader) (heartbeat HeartbeatData, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if err != io.EOF {
				logger.Sugar().Info("error from parseHeartbeatData 1 err: ", err)
				err = errors.Wrapf(errs.ErrGT06BadDataPacket, "from parseHeartbeatData")
			}
		}
	}()

	var terminalInfoByte byte
	if err := binary.Read(reader, binary.BigEndian, &terminalInfoByte); err != nil {
		return heartbeat, err
	}
	logger.Sugar().Infof("parseHeartbeatData Terminal Info Byte: %x", terminalInfoByte)
	heartbeat.TerminalInformation, err = p.parseTerminalInfoFromByte(terminalInfoByte)
	if err != nil {
		return heartbeat, err
	}

	var batteryLevelByte byte
	if err := binary.Read(reader, binary.BigEndian, &batteryLevelByte); err != nil {
		return heartbeat, err
	}
	logger.Sugar().Infof("parseHeartbeatData  Battery Level Byte: %x", batteryLevelByte)
	heartbeat.BatteryLevel = BatteryLevel(batteryLevelByte)
	if heartbeat.BatteryLevel == VL_Invalid {
		return heartbeat, errs.ErrGT06InvalidVoltageLevel
	}

	var gsmSignalStrengthByte byte
	if err := binary.Read(reader, binary.BigEndian, &gsmSignalStrengthByte); err != nil {
		return heartbeat, err
	}
	logger.Sugar().Infof("parseHeartbeatData GSM Signal Strength Byte: %x", gsmSignalStrengthByte)
	heartbeat.GSMSignalStrength = GSMSignalStrength(gsmSignalStrengthByte)
	if heartbeat.GSMSignalStrength == GSM_Invalid {
		return heartbeat, errs.ErrGT06InvalidGSMSignalStrength
	}

	if err := binary.Read(reader, binary.BigEndian, &heartbeat.ExtendedPortStatus); err != nil {
		return heartbeat, err
	}
	logger.Sugar().Infof("parseHeartbeatData Extended Port Status Byte: %x", heartbeat.ExtendedPortStatus)

	if _, err := reader.Peek(1); err != io.EOF {
		logger.Sugar().Errorf("parseHeartbeatData Extra bytes detected in packet")
		logger.Sugar().Info("error from parseHeartbeatData 2")
		return heartbeat, errors.Wrapf(errs.ErrGT06BadDataPacket, "from parseHeartbeatData 2")
	}

	return heartbeat, nil
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

	var i32 uint32
	// latitude
	checkErr(binary.Read(reader, binary.BigEndian, &i32))
	gpsInfo.Latitude = float32(i32) / 1800000

	// longitude
	checkErr(binary.Read(reader, binary.BigEndian, &i32))
	gpsInfo.Longitude = float32(i32) / 1800000

	// speed
	checkErr(binary.Read(reader, binary.BigEndian, &gpsInfo.Speed))
	logger.Sugar().Info("speed from parseGPSInformation: ", gpsInfo.Speed)

	// TODO: parse the 16-bit course to detailed fields
	// course/heading
	var courseValue uint16
	checkErr(binary.Read(reader, binary.BigEndian, &courseValue))
	gpsInfo.Course = p.parseGpsCourse(courseValue)

	return
}

func (p *TR06Protocol) parseGpsCourse(courseValue uint16) (course GPSCourse) {
	b1 := byte(courseValue >> 8)

	course.IsRealtime = b1&0x20 == 0x00     // byte 1, bit 5 is 0
	course.IsDifferential = b1&0x20 == 0x20 // byte 1, bit 5 is 1
	course.Positioned = b1&0x10 == 0x10     // byte 1, bit 4 is 0
	course.Longitude = b1&0x08 == 0x08      // byte 1, bit 3 is 0
	course.Latitude = b1&0x04 == 0x04       // byte 1, bit 2 is 0

	course.Degree = courseValue & 0x03ff // byte 1 (bit 1, 0), byte 2
	return
}

func (p *TR06Protocol) parseTimestamp(reader *bufio.Reader) (timestamp time.Time, err error) {
	year, err := reader.ReadByte()
	checkErr(err)

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

	var timezone = time.UTC

	timestamp = time.Date(int(year), time.Month(month), int(day), int(hour), int(minute), int(second), 0, timezone)
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
	checkErr(binary.Read(reader, binary.BigEndian, &statusInfo.GSMSignalStrength))
	if statusInfo.GSMSignalStrength == GSM_Invalid {
		return statusInfo, errs.ErrGT06InvalidGSMSignalStrength
	}

	// alarm status
	alarm, err := reader.ReadByte()
	checkErr(err)
	statusInfo.Alarm = AlarmValue(alarm)

	language, err := reader.ReadByte()
	checkErr(err)
	statusInfo.Language = Language(language)
	return
}

func (p *TR06Protocol) parseTerminalInfoFromByte(terminalInfoByte byte) (TerminalInformation, error) {
	var terminalInfo TerminalInformation
	terminalInfo.OilElectricityConnected = terminalInfoByte&0x80 == 0x80 // bit 7
	terminalInfo.GPSSignalAvailable = terminalInfoByte&0x40 == 0x40      // bit 6
	terminalInfo.AlarmType = AlarmType(terminalInfoByte & 0x38)          // bit 3, 4, 5
	terminalInfo.Charging = terminalInfoByte&0x10 == 0x08                // bit 2
	terminalInfo.ACCHigh = terminalInfoByte&0x20 == 0x02                 // bit 1
	terminalInfo.Armed = terminalInfoByte&0x01 == 0x01                   // bit 0

	if terminalInfo.AlarmType == AL_Invalid {
		return terminalInfo, errs.ErrGT06InvalidAlarmType
	}
	return terminalInfo, nil
}

func (p *TR06Protocol) IsValidHeader(reader *bufio.Reader) bool {
	header, err := reader.Peek(2)
	if err != nil {
		return false
	}

	if bytes.Equal(header, []byte{0x78, 0x78}) || bytes.Equal(header, []byte{0x79, 0x79}) {
		return true
	}
	return false
}
