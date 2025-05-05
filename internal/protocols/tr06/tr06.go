package tr06

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"slices"

	"github.com/404minds/avl-receiver/internal/crc"
	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var logger = configuredLogger.Logger

type TR06Protocol struct {
	LoginInformation *LoginData
	DeviceType       types.DeviceType
}

func (p *TR06Protocol) GetDeviceID() string {
	logger.Sugar().Infoln("GetDeviceID: start")
	if p.LoginInformation == nil {
		logger.Sugar().Infoln("GetDeviceID: step 1 - LoginInformation is nil")
		logger.Error("LoginInformation is nil in GetDeviceID")
		return ""
	}
	logger.Sugar().Infoln("GetDeviceID: step 2 - have LoginInformation")

	if p.LoginInformation.TerminalID == "" {
		logger.Sugar().Infoln("GetDeviceID: step 3 - TerminalID is empty")
		logger.Error("Login Information does not have TerminalID in GetDeviceID")
	}
	logger.Sugar().Infoln("GetDeviceID: end, returning", p.LoginInformation.TerminalID)
	return p.LoginInformation.TerminalID
}

func (p *TR06Protocol) GetDeviceType() types.DeviceType {
	logger.Sugar().Infoln("GetDeviceType: start & end")
	return p.DeviceType
}

func (p *TR06Protocol) SetDeviceType(t types.DeviceType) {
	logger.Sugar().Infoln("SetDeviceType: setting to", t)
	p.DeviceType = t
}

func (p *TR06Protocol) GetProtocolType() types.DeviceProtocolType {
	logger.Sugar().Infoln("GetProtocolType: start & end")
	return types.DeviceProtocolType_GT06
}

func (p *TR06Protocol) Login(reader *bufio.Reader) (ack []byte, byteToSkip int, e error) {
	logger.Sugar().Infoln("Login: start")
	logger.Sugar().Infoln("Login: step 1 - validating header")
	if !p.IsValidHeader(reader) {
		logger.Sugar().Infoln("Login: step 1 failed - invalid header")
		return nil, 0, errs.ErrUnknownProtocol
	}
	logger.Sugar().Infoln("Login: step 1 ok - valid header")

	logger.Sugar().Infoln("Login: step 2 - peeking available data")
	data, _ := reader.Peek(reader.Buffered())
	logger.Sugar().Infoln("Login: available data before reading IMEI:", data)

	logger.Sugar().Infoln("Login: step 3 - parsing packet")
	packet, err := p.parsePacket(reader)
	logger.Sugar().Infoln("Login: parsePacket returned err =", err)
	if err != nil {
		logger.Error("Login: failed to parse GT06 packet", zap.Error(err))
		return nil, 0, err
	}
	logger.Sugar().Infoln("Login: step 3 ok - got packet with MessageType", packet.MessageType)

	if packet.MessageType != MSG_LoginData {
		logger.Sugar().Infoln("Login: step 4 failed - not a login packet, type =", packet.MessageType)
		logger.Error("packet message type is not MSG_LoginData", zap.Error(errs.ErrGT06InvalidLoginInfo))
		return nil, 0, errs.ErrGT06InvalidLoginInfo
	}
	logger.Sugar().Infoln("Login: step 4 ok - MSG_LoginData")

	logger.Sugar().Infoln("Login: step 5 - validating packet.Information")
	if packet.Information == nil {
		logger.Sugar().Infoln("Login: step 5 failed - Information is nil")
		logger.Error("packet information is nil", zap.Error(errs.ErrGT06InvalidLoginInfo))
		return nil, 0, errs.ErrGT06InvalidLoginInfo
	}

	loginData, ok := packet.Information.(*LoginData)
	logger.Sugar().Infoln("Login: step 6 - type assertion to *LoginData ok?", ok)
	if !ok || loginData == nil {
		logger.Sugar().Infoln("Login: step 6 failed - invalid loginData")
		logger.Error("packet information is not of type *LoginData", zap.Error(errs.ErrGT06InvalidLoginInfo))
		return nil, 0, errs.ErrGT06InvalidLoginInfo
	}
	logger.Sugar().Infoln("Login: step 6 ok - got loginData", loginData)

	p.LoginInformation = loginData
	logger.Sugar().Infoln("Login: step 7 - stored LoginInformation in protocol")

	logger.Sugar().Infoln("Login: step 8 - preparing response")
	byteBuffer := bytes.NewBuffer([]byte{})
	err = p.sendResponse(packet, byteBuffer)
	logger.Sugar().Infoln("Login: sendResponse returned err =", err)
	if err != nil {
		logger.Error("failed to send response for GT06 packet", zap.Error(err))
		return nil, 0, err
	}

	logger.Sugar().Infoln("Login: end, returning ack bytes")
	return byteBuffer.Bytes(), 0, nil
}

func (p *TR06Protocol) ConsumeStream(reader *bufio.Reader, writer io.Writer, dataStore store.Store) error {
	logger.Sugar().Infoln("ConsumeStream: start")
	for {
		logger.Sugar().Infoln("ConsumeStream: step 1 - parsing packet")
		packet, err := p.parsePacket(reader)
		logger.Sugar().Infoln("ConsumeStream: parsePacket returned err =", err)
		if err != nil {
			logger.Sugar().Infoln("ConsumeStream: step 1 failed, returning err")
			return err
		}

		logger.Sugar().Infoln("ConsumeStream: step 2 - checking for heartbeat")
		if packet.MessageType == MSG_HeartbeatData {
			logger.Sugar().Infoln("ConsumeStream: step 2 ok - heartbeat, sending response")
			err = p.sendResponse(packet, writer)
			logger.Sugar().Infoln("ConsumeStream: sendResponse returned err =", err)
			if err != nil {
				logger.Sugar().Infoln("ConsumeStream: step 2 failed to respond, returning err")
				return err
			}
		}

		logger.Sugar().Infoln("ConsumeStream: step 3 - forwarding to dataStore")
		asyncStore := dataStore.GetProcessChan()
		protoPacket := packet.ToProtobufDeviceStatus(p.GetDeviceID(), p.DeviceType)
		asyncStore <- protoPacket
		logger.Sugar().Infoln("ConsumeStream: forwarded packet to channel")
	}
}

func (p *TR06Protocol) sendResponse(parsedPacket *Packet, writer io.Writer) error {
	logger.Sugar().Infoln("sendResponse: start")
	defer func() {
		if condition := recover(); condition != nil {
			logger.Error("Failed to write response packet", zap.Any("panic", condition))
		}
	}()

	logger.Sugar().Infoln("sendResponse: step 1 - building ResponsePacket struct")
	responsePacket := ResponsePacket{
		StartBit:                0x7878,
		PacketLength:            0x05,
		ProtocolNumber:          int8(parsedPacket.MessageType),
		InformationSerialNumber: parsedPacket.InformationSerialNumber,
		StopBits:                0xd0a,
	}

	logger.Sugar().Infoln("sendResponse: step 2 - calculating CRC")
	responsePacket.Crc = crc.CrcWanway(responsePacket.ToBytes()[2:6])

	logger.Sugar().Infoln("sendResponse: step 3 - writing bytes to writer")
	bytesToSend := responsePacket.ToBytes()
	logger.Sugar().Infoln("sendResponse: bytes =", bytesToSend)
	_, err := writer.Write(bytesToSend)
	logger.Sugar().Infoln("sendResponse: write returned err =", err)
	if err != nil {
		return errors.Wrapf(err, "failed to write response packet")
	}
	logger.Sugar().Infoln("sendResponse: end success")
	return nil
}

func (p *TR06Protocol) parsePacket(reader *bufio.Reader) (packet *Packet, err error) {
	logger.Sugar().Infoln("parsePacket: start")
	defer func() {
		if r := recover(); r != nil {
			logger.Sugar().Infoln("parsePacket: panic recovered")
			if rErr, ok := r.(error); ok {
				err = rErr
			} else {
				err = fmt.Errorf("parse packet unknown panic: %v", r)
			}
			if err != io.EOF {
				err = errors.Wrapf(errs.ErrGT06BadDataPacket, "from parsePacket")
			}
			logger.Sugar().Errorf("parsePacket recovered: %v", err)
		}
	}()

	packet = &Packet{}

	logger.Sugar().Infoln("parsePacket: step 1 - reading StartBit")
	if err = binary.Read(reader, binary.BigEndian, &packet.StartBit); err != nil {
		logger.Sugar().Errorf("parsePacket: step 1 failed to read start bit: %v", err)
		return nil, err
	}
	logger.Sugar().Infoln("parsePacket: StartBit =", fmt.Sprintf("0x%x", packet.StartBit))

	logger.Sugar().Infoln("parsePacket: step 2 - determining packet length")
	if packet.StartBit == 0x7878 {
		var packetLength byte
		if err = binary.Read(reader, binary.BigEndian, &packetLength); err != nil {
			logger.Sugar().Errorf("parsePacket: failed to read packet length: %v", err)
			return nil, err
		}
		packet.PacketLength = packetLength
		logger.Sugar().Infoln("parsePacket: PacketLength =", packetLength)
	} else if packet.StartBit == 0x7979 {
		// handle 0x7979 if needed
		logger.Sugar().Infoln("parsePacket: encountered 0x7979 start bit, skipping custom logic")
	} else {
		logger.Sugar().Infoln("parsePacket: invalid start bit, returning ErrGT06BadDataPacket")
		return nil, errors.Wrapf(errs.ErrGT06BadDataPacket, "Invalid StartBit: %x", packet.StartBit)
	}

	logger.Sugar().Infoln("parsePacket: step 3 - reading packetData (length-4 bytes)")
	packetData := make([]byte, packet.PacketLength-4)
	if _, err = io.ReadFull(reader, packetData); err != nil {
		logger.Sugar().Errorf("parsePacket: failed to read packet data: %v", err)
		return nil, err
	}
	logger.Sugar().Infoln("parsePacket: packetData =", packetData)

	logger.Sugar().Infoln("parsePacket: step 4 - parsing packetData fields")
	if err = p.parsePacketData(bufio.NewReader(bytes.NewReader(packetData)), packet); err != nil {
		logger.Sugar().Errorf("parsePacket: parsePacketData failed: %v", err)
		return nil, err
	}

	logger.Sugar().Infoln("parsePacket: step 5 - reading InformationSerialNumber")
	if err = binary.Read(reader, binary.BigEndian, &packet.InformationSerialNumber); err != nil {
		logger.Sugar().Errorf("parsePacket: failed to read information serial number: %v", err)
		return nil, err
	}
	logger.Sugar().Infoln("parsePacket: InformationSerialNumber =", packet.InformationSerialNumber)

	logger.Sugar().Infoln("parsePacket: step 6 - reading CRC")
	if err = binary.Read(reader, binary.BigEndian, &packet.Crc); err != nil {
		logger.Sugar().Errorf("parsePacket: failed to read CRC: %v", err)
		return nil, err
	}
	logger.Sugar().Infoln("parsePacket: CRC =", fmt.Sprintf("0x%x", packet.Crc))

	logger.Sugar().Infoln("parsePacket: step 7 - reading StopBits")
	if err = binary.Read(reader, binary.BigEndian, &packet.StopBits); err != nil {
		logger.Sugar().Errorf("parsePacket: failed to read stop bits: %v", err)
		return nil, err
	}
	logger.Sugar().Infoln("parsePacket: StopBits =", fmt.Sprintf("0x%x", packet.StopBits))

	if packet.StopBits != 0x0d0a {
		logger.Sugar().Errorf("parsePacket: invalid stop bits: %x", packet.StopBits)
		return nil, errors.Wrapf(errs.ErrGT06BadDataPacket, "Invalid StopBits")
	}

	logger.Sugar().Infoln("parsePacket: step 8 - validating CRC")
	expectedCrc := crc.CrcWanway(
		slices.Concat(
			[]byte{packet.PacketLength},
			packetData,
			[]byte{
				byte(packet.InformationSerialNumber >> 8),
				byte(packet.InformationSerialNumber & 0xff),
			},
		),
	)
	if expectedCrc != packet.Crc {
		logger.Sugar().Errorf("parsePacket: CRC mismatch, expected %x got %x", expectedCrc, packet.Crc)
		return nil, errs.ErrBadCrc
	}

	logger.Sugar().Infoln("parsePacket: end, returning packet")
	return packet, nil
}

func (p *TR06Protocol) parsePacketData(reader *bufio.Reader, packet *Packet) error {
	logger.Sugar().Infoln("parsePacketData: start")
	logger.Sugar().Infoln("parsePacketData: step 1 - reading protocol number byte")
	protocolNumByte, err := reader.ReadByte()
	logger.Sugar().Infoln("parsePacketData: protocol number byte =", protocolNumByte)
	if err != nil {
		return err
	}

	msgType := MessageType(protocolNumByte)
	packet.MessageType = msgType
	logger.Sugar().Infoln("parsePacketData: step 2 - messageType =", msgType)

	if msgType == MSG_Invalid {
		logger.Sugar().Infoln("parsePacketData: step 3 - invalid message type")
		remainingData, err2 := p.consumePacket(reader)
		if err2 != nil {
			return err2
		}
		logger.Sugar().Infoln("parsePacketData: dumped invalid bytes:", hex.Dump(remainingData))
		return errors.Wrapf(errs.ErrGT06BadDataPacket, "Invalid message type")
	}

	logger.Sugar().Infoln("parsePacketData: step 4 - parsing packet information")
	if packet.Information, err = p.parsePacketInformation(reader, packet.MessageType); err != nil {
		logger.Sugar().Infoln("parsePacketData: parsePacketInformation failed:", err)
		return err
	}
	logger.Sugar().Infoln("parsePacketData: end")
	return nil
}

func (p *TR06Protocol) consumePacket(reader *bufio.Reader) ([]byte, error) {
	logger.Sugar().Infoln("consumePacket: start")
	data := make([]byte, 0)
	term := []byte{0x0d, 0x0a}

	for {
		b, err := reader.ReadByte()
		if err != nil {
			logger.Sugar().Infoln("consumePacket: read error", err)
			return nil, err
		}
		data = append(data, b)
		if bytes.HasSuffix(data, term) {
			break
		}
	}
	logger.Sugar().Infoln("consumePacket: end, data =", data)
	return data, nil
}

func (p *TR06Protocol) parsePacketInformation(reader *bufio.Reader, messageType MessageType) (interface{}, error) {
	logger.Sugar().Infoln("parsePacketInformation: start, messageType =", messageType)
	switch messageType {
	case MSG_LoginData:
		return p.parseLoginInformation(reader)
	case MSG_PositioningData:
		return p.parsePositioningData(reader)
	case MSG_AlarmData:
		return p.parseAlarmData(reader)
	case MSG_HeartbeatData, MSG_EG_HeartbeatData:
		return p.parseHeartbeatData(reader)
	case MSG_TransmissionInstruction:
		return p.parseInformationTransmissionPacket(reader)
	default:
		logger.Sugar().Infoln("parsePacketInformation: unknown messageType, returning error")
		return nil, errors.Wrapf(errs.ErrGT06BadDataPacket, "unknown message type")
	}
}

func (p *TR06Protocol) parseLoginInformation(reader *bufio.Reader) (interface{}, error) {
	logger.Sugar().Infoln("parseLoginInformation: start")
	var loginInfo LoginData

	logger.Sugar().Infoln("parseLoginInformation: step 1 - reading IMEI bytes")
	var imeiBytes [8]byte
	if err := binary.Read(reader, binary.BigEndian, &imeiBytes); err != nil {
		logger.Error("failed to read IMEI bytes", zap.Error(err))
		return nil, errs.ErrGT06InvalidLoginInfo
	}
	logger.Sugar().Infoln("parseLoginInformation: imeiBytes =", imeiBytes[:])

	loginInfo.TerminalID = hex.EncodeToString(imeiBytes[:])[1:]
	logger.Sugar().Infoln("parseLoginInformation: TerminalID =", loginInfo.TerminalID)

	logger.Sugar().Infoln("parseLoginInformation: end")
	return &loginInfo, nil
}

func (p *TR06Protocol) parsePositioningData(reader *bufio.Reader) (interface{}, error) {
	logger.Sugar().Infoln("parsePositioningData: start")
	defer func() {
		if r := recover(); r != nil {
			logger.Sugar().Infoln("parsePositioningData: panic", r)
		}
	}()

	var parsed PositioningInformation

	logger.Sugar().Infoln("parsePositioningData: step 1 - parsing timestamp")
	timestamp, err := p.parseTimestamp(reader)
	if err != nil {
		logger.Sugar().Errorf("parsePositioningData: failed to parse timestamp: %v", err)
		return nil, errors.Wrap(err, "failed to parse timestamp")
	}
	parsed.GpsInformation.Timestamp = timestamp
	logger.Sugar().Infoln("parsePositioningData: timestamp =", timestamp)

	logger.Sugar().Infoln("parsePositioningData: step 2 - reading GPS info and satellite count")
	var gpsInfo byte
	if err := binary.Read(reader, binary.BigEndian, &gpsInfo); err != nil {
		return nil, err
	}
	parsed.GpsInformation.GPSInfoLength = gpsInfo >> 4
	parsed.GpsInformation.NumberOfSatellites = gpsInfo & 0x0F
	logger.Sugar().Infoln("parsePositioningData: GPSInfoLength =", parsed.GpsInformation.GPSInfoLength,
		"NumSatellites =", parsed.GpsInformation.NumberOfSatellites)

	logger.Sugar().Infoln("parsePositioningData: step 3 - reading latitude")
	var latitude uint32
	if err := binary.Read(reader, binary.BigEndian, &latitude); err != nil {
		return nil, err
	}
	parsed.GpsInformation.Latitude = float32(latitude) / 30000 / 60
	logger.Sugar().Infoln("parsePositioningData: Latitude =", parsed.GpsInformation.Latitude)

	logger.Sugar().Infoln("parsePositioningData: step 4 - reading longitude")
	var longitude uint32
	if err := binary.Read(reader, binary.BigEndian, &longitude); err != nil {
		return nil, err
	}
	parsed.GpsInformation.Longitude = float32(longitude) / 30000 / 60
	logger.Sugar().Infoln("parsePositioningData: Longitude =", parsed.GpsInformation.Longitude)

	logger.Sugar().Infoln("parsePositioningData: step 5 - reading speed")
	if err := binary.Read(reader, binary.BigEndian, &parsed.GpsInformation.Speed); err != nil {
		return nil, err
	}
	logger.Sugar().Infoln("parsePositioningData: Speed =", parsed.GpsInformation.Speed)

	logger.Sugar().Infoln("parsePositioningData: step 6 - reading course and status")
	var courseAndStatus [2]byte
	if err := binary.Read(reader, binary.BigEndian, &courseAndStatus); err != nil {
		return nil, err
	}
	parsed.GpsInformation.Course = parseCourseAndStatus(courseAndStatus)
	logger.Sugar().Infoln("parsePositioningData: Course =", parsed.GpsInformation.Course)

	logger.Sugar().Infoln("parsePositioningData: step 7 - reading LBS info")
	if err := binary.Read(reader, binary.BigEndian, &parsed.LBSInfo.MCC); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.BigEndian, &parsed.LBSInfo.MNC); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.BigEndian, &parsed.LBSInfo.LAC); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.BigEndian, &parsed.LBSInfo.CellID); err != nil {
		return nil, err
	}
	logger.Sugar().Infoln("parsePositioningData: LBSInfo =", parsed.LBSInfo)

	logger.Sugar().Infoln("parsePositioningData: end")
	return &parsed, nil
}

func parseCourseAndStatus(courseAndStatus [2]byte) GPSCourse {
	// no additional logs here; it’s a pure helper
	var course GPSCourse
	course.IsRealtime = (courseAndStatus[0] & 0x40) == 0
	course.IsDifferential = (courseAndStatus[0] & 0x20) != 0
	course.Positioned = (courseAndStatus[0] & 0x10) != 0
	course.Longitude = (courseAndStatus[0] & 0x08) != 0
	course.Latitude = (courseAndStatus[0] & 0x04) != 0
	course.Degree = uint16(courseAndStatus[1]) | (uint16(courseAndStatus[0]&0x03) << 8)
	return course
}

func (p *TR06Protocol) parseAlarmData(reader *bufio.Reader) (alarmInfo AlarmInformation, err error) {
	logger.Sugar().Infoln("parseAlarmData: start")
	defer func() {
		if r := recover(); r != nil {
			logger.Sugar().Infoln("parseAlarmData: panic", r)
		}
	}()

	logger.Sugar().Infoln("parseAlarmData: step 1 - parsing GPS information")
	alarmInfo.GpsInformation, err = p.parseGPSInformation(reader)
	if err != nil {
		return alarmInfo, err
	}

	logger.Sugar().Infoln("parseAlarmData: step 2 - parsing LBS information")
	alarmInfo.LBSInformation, err = p.parseLBSInformation(reader)
	if err != nil {
		return alarmInfo, err
	}

	logger.Sugar().Infoln("parseAlarmData: step 3 - parsing Status information")
	alarmInfo.StatusInformation, err = p.parseStatusInformation(reader)
	if err != nil {
		return alarmInfo, err
	}

	logger.Sugar().Infoln("parseAlarmData: end")
	return
}

func (p *TR06Protocol) parseHeartbeatData(reader *bufio.Reader) (heartbeat HeartbeatData, err error) {
	logger.Sugar().Infoln("parseHeartbeatData: start")
	defer func() {
		if r := recover(); r != nil {
			logger.Sugar().Infoln("parseHeartbeatData: panic", r)
		}
	}()

	logger.Sugar().Infoln("parseHeartbeatData: step 1 - reading terminalInfoByte")
	var terminalInfoByte byte
	if err = binary.Read(reader, binary.BigEndian, &terminalInfoByte); err != nil {
		return
	}
	logger.Sugar().Infoln("parseHeartbeatData: terminalInfoByte =", terminalInfoByte)

	logger.Sugar().Infoln("parseHeartbeatData: step 2 - parsing terminal info")
	heartbeat.TerminalInformation, err = p.parseTerminalInfoFromByte(terminalInfoByte)
	if err != nil {
		return
	}

	logger.Sugar().Infoln("parseHeartbeatData: step 3 - reading batteryLevelByte")
	var batteryLevelByte byte
	if err = binary.Read(reader, binary.BigEndian, &batteryLevelByte); err != nil {
		return
	}
	logger.Sugar().Infoln("parseHeartbeatData: batteryLevelByte =", batteryLevelByte)
	heartbeat.BatteryLevel = BatteryLevel(batteryLevelByte)
	if heartbeat.BatteryLevel == VL_Invalid {
		return heartbeat, errs.ErrGT06InvalidVoltageLevel
	}

	logger.Sugar().Infoln("parseHeartbeatData: step 4 - reading GSM signal strength")
	var gsmSignalStrengthByte byte
	if err = binary.Read(reader, binary.BigEndian, &gsmSignalStrengthByte); err != nil {
		return
	}
	logger.Sugar().Infoln("parseHeartbeatData: gsmSignalStrengthByte =", gsmSignalStrengthByte)
	heartbeat.GSMSignalStrength = GSMSignalStrength(gsmSignalStrengthByte)
	if heartbeat.GSMSignalStrength == GSM_Invalid {
		return heartbeat, errs.ErrGT06InvalidGSMSignalStrength
	}

	logger.Sugar().Infoln("parseHeartbeatData: step 5 - reading ExtendedPortStatus")
	if err = binary.Read(reader, binary.BigEndian, &heartbeat.ExtendedPortStatus); err != nil {
		return
	}
	logger.Sugar().Infoln("parseHeartbeatData: ExtendedPortStatus =", heartbeat.ExtendedPortStatus)

	logger.Sugar().Infoln("parseHeartbeatData: step 6 - checking for extra bytes")
	if _, err2 := reader.Peek(1); err2 != io.EOF {
		logger.Sugar().Errorf("parseHeartbeatData: extra bytes detected")
		return heartbeat, errors.Wrapf(errs.ErrGT06BadDataPacket, "Extra bytes in heartbeat")
	}

	logger.Sugar().Infoln("parseHeartbeatData: end")
	return heartbeat, nil
}

func (p *TR06Protocol) parseInformationTransmissionPacket(reader *bufio.Reader) (packet InformationTransmissionPacket, err error) {
	logger.Sugar().Infoln("parseInformationTransmissionPacket: start")
	defer func() {
		if r := recover(); r != nil {
			logger.Sugar().Infoln("parseInformationTransmissionPacket: panic", r)
		}
	}()

	logger.Sugar().Infoln("parseInformationTransmissionPacket: step 1 - reading informationType")
	var informationType byte
	if err = binary.Read(reader, binary.BigEndian, &informationType); err != nil {
		return
	}
	logger.Sugar().Infoln("parseInformationTransmissionPacket: informationType =", informationType)
	packet.InformationContent.InformationType = InformationType(informationType)

	logger.Sugar().Infoln("parseInformationTransmissionPacket: step 2 - reading 2-byte dataContent")
	dataContent := make([]byte, 2)
	if _, err = io.ReadFull(reader, dataContent); err != nil {
		return
	}
	logger.Sugar().Infoln("parseInformationTransmissionPacket: dataContent =", dataContent)

	logger.Sugar().Infoln("parseInformationTransmissionPacket: step 3 - interpreting based on InformationType")
	switch InformationType(informationType) {
	case ExternalPowerVoltage:
		if len(dataContent) < 2 {
			return packet, errors.New("Insufficient data for ExternalPowerVoltage")
		}
		voltage := binary.BigEndian.Uint16(dataContent)
		packet.InformationContent.DataContent = voltage / 100
	case TerminalStatusSync, DoorStatus:
		// no-op or one-byte
	default:
		logger.Sugar().Infoln("parseInformationTransmissionPacket: unknown type, skipping dataContent logic")
	}

	logger.Sugar().Infoln("parseInformationTransmissionPacket: step 4 - checking for extra bytes")
	if remain, err2 := reader.Peek(1); err2 != io.EOF {
		logger.Sugar().Errorf("parseInformationTransmissionPacket: extra bytes %x", remain)
		return packet, errors.New("GT06 Bad Data Packet")
	}

	logger.Sugar().Infoln("parseInformationTransmissionPacket: end")
	return packet, nil
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func (p *TR06Protocol) parseGPSInformation(reader *bufio.Reader) (gpsInfo GPSInformation, err error) {
	logger.Sugar().Infoln("parseGPSInformation: start")
	// reuse timestamp parsing logs
	gpsInfo.Timestamp, err = p.parseTimestamp(reader)
	checkErr(err)

	logger.Sugar().Infoln("parseGPSInformation: step 2 - reading GPSInfoLength/sats")
	var x byte
	x, err = reader.ReadByte()
	checkErr(err)
	gpsInfo.GPSInfoLength = x >> 4
	gpsInfo.NumberOfSatellites = x & 0x0f
	logger.Sugar().Infoln("parseGPSInformation: GPSInfoLength/sats =", gpsInfo.GPSInfoLength, gpsInfo.NumberOfSatellites)

	logger.Sugar().Infoln("parseGPSInformation: step 3 - reading latitude")
	var i32 int32
	checkErr(binary.Read(reader, binary.BigEndian, &i32))
	gpsInfo.Latitude = float32(i32) / 1800000

	logger.Sugar().Infoln("parseGPSInformation: step 4 - reading longitude")
	checkErr(binary.Read(reader, binary.BigEndian, &i32))
	gpsInfo.Longitude = float32(i32) / 1800000

	logger.Sugar().Infoln("parseGPSInformation: step 5 - reading speed")
	checkErr(binary.Read(reader, binary.BigEndian, &gpsInfo.Speed))

	logger.Sugar().Infoln("parseGPSInformation: step 6 - reading course value")
	var courseValue uint16
	checkErr(binary.Read(reader, binary.BigEndian, &courseValue))
	gpsInfo.Course = p.parseGpsCourse(courseValue)

	logger.Sugar().Infoln("parseGPSInformation: end, result =", gpsInfo)
	return
}

func (p *TR06Protocol) parseGpsCourse(courseValue uint16) GPSCourse {
	// helper, no logs
	var course GPSCourse
	b1 := byte(courseValue >> 8)
	course.IsRealtime = b1&0x20 == 0
	course.IsDifferential = b1&0x20 != 0
	course.Positioned = b1&0x10 != 0
	course.Longitude = b1&0x08 != 0
	course.Latitude = b1&0x04 != 0
	course.Degree = courseValue & 0x03ff
	return course
}

func (p *TR06Protocol) parseTimestamp(reader *bufio.Reader) (timestamp time.Time, err error) {
	logger.Sugar().Infoln("parseTimestamp: start")
	var (
		yearB  byte
		monthB byte
		dayB   byte
		hourB  byte
		minB   byte
		secB   byte
	)
	checkErr(binary.Read(reader, binary.BigEndian, &yearB))
	checkErr(binary.Read(reader, binary.BigEndian, &monthB))
	checkErr(binary.Read(reader, binary.BigEndian, &dayB))
	checkErr(binary.Read(reader, binary.BigEndian, &hourB))
	checkErr(binary.Read(reader, binary.BigEndian, &minB))
	checkErr(binary.Read(reader, binary.BigEndian, &secB))

	year := int(yearB) + 2000
	logger.Sugar().Infoln("parseTimestamp: components =", year, monthB, dayB, hourB, minB, secB)
	timestamp = time.Date(year, time.Month(monthB), int(dayB), int(hourB), int(minB), int(secB), 0, time.UTC)
	logger.Sugar().Infoln("parseTimestamp: end, timestamp =", timestamp)
	return
}

func (p *TR06Protocol) parseLBSInformation(reader *bufio.Reader) (lbsInfo LBSInformation, err error) {
	logger.Sugar().Infoln("parseLBSInformation: start")
	checkErr(binary.Read(reader, binary.BigEndian, &lbsInfo.MCC))
	checkErr(binary.Read(reader, binary.BigEndian, &lbsInfo.MNC))
	checkErr(binary.Read(reader, binary.BigEndian, &lbsInfo.LAC))
	checkErr(binary.Read(reader, binary.BigEndian, &lbsInfo.CellID))
	logger.Sugar().Infoln("parseLBSInformation: end, LBSInfo =", lbsInfo)
	return
}

func (p *TR06Protocol) parseStatusInformation(reader *bufio.Reader) (statusInfo StatusInformation, err error) {
	logger.Sugar().Infoln("parseStatusInformation: start")
	var b byte

	checkErr(binary.Read(reader, binary.BigEndian, &b))
	statusInfo.TerminalInformation, err = p.parseTerminalInfoFromByte(b)
	checkErr(err)
	logger.Sugar().Infoln("parseStatusInformation: TerminalInformation =", statusInfo.TerminalInformation)

	checkErr(binary.Read(reader, binary.BigEndian, &b))
	statusInfo.BatteryLevel = BatteryLevel(b)
	if statusInfo.BatteryLevel == VL_Invalid {
		return statusInfo, errs.ErrGT06InvalidAlarmType
	}
	logger.Sugar().Infoln("parseStatusInformation: BatteryLevel =", statusInfo.BatteryLevel)

	checkErr(binary.Read(reader, binary.BigEndian, &b))
	statusInfo.GSMSignalStrength = GSMSignalStrength(b)
	checkErr(nil)
	if statusInfo.GSMSignalStrength == GSM_Invalid {
		return statusInfo, errs.ErrGT06InvalidGSMSignalStrength
	}
	logger.Sugar().Infoln("parseStatusInformation: GSMSignalStrength =", statusInfo.GSMSignalStrength)

	checkErr(binary.Read(reader, binary.BigEndian, &b))
	statusInfo.Alarm = AlarmValue(b)
	checkErr(binary.Read(reader, binary.BigEndian, &b))
	statusInfo.Language = Language(b)
	logger.Sugar().Infoln("parseStatusInformation: Alarm =", statusInfo.Alarm, "Language =", statusInfo.Language)

	logger.Sugar().Infoln("parseStatusInformation: end")
	return
}

func (p *TR06Protocol) parseTerminalInfoFromByte(terminalInfoByte byte) (TerminalInformation, error) {
	logger.Sugar().Infoln("parseTerminalInfoFromByte: start, byte =", terminalInfoByte)
	var ti TerminalInformation
	ti.OilElectricityConnected = terminalInfoByte&0x80 == 0x80
	ti.GPSSignalAvailable = terminalInfoByte&0x40 == 0x40
	ti.AlarmType = AlarmType(terminalInfoByte & 0x38)
	ti.Charging = terminalInfoByte&0x10 == 0x08
	ti.ACCHigh = terminalInfoByte&0x20 == 0x02
	ti.Armed = terminalInfoByte&0x01 == 0x01
	if ti.AlarmType == AL_Invalid {
		logger.Sugar().Infoln("parseTerminalInfoFromByte: invalid AlarmType")
		return ti, errs.ErrGT06InvalidAlarmType
	}
	logger.Sugar().Infoln("parseTerminalInfoFromByte: end, TerminalInformation =", ti)
	return ti, nil
}

func (p *TR06Protocol) IsValidHeader(reader *bufio.Reader) bool {
	logger.Sugar().Infoln("IsValidHeader: start")
	header, err := reader.Peek(2)
	if err != nil {
		logger.Sugar().Infoln("IsValidHeader: peek error", err)
		return false
	}
	ok := bytes.Equal(header, []byte{0x78, 0x78}) || bytes.Equal(header, []byte{0x79, 0x79})
	logger.Sugar().Infoln("IsValidHeader: header =", header, "valid?", ok)
	return ok
}

func (p *TR06Protocol) SendCommandToDevice(writer io.Writer, command string) error {
	logger.Sugar().Infoln("SendCommandToDevice: start, command =", command)
	// TODO: implement sending HEX of command
	logger.Sugar().Infoln("SendCommandToDevice: end (not implemented)")
	return nil
}
