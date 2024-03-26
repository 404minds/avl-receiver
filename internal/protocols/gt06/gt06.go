package gt06

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"go.uber.org/zap"
	"io"
	"slices"
	"time"

	"github.com/404minds/avl-receiver/internal/crc"
	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/types"
)

var logger = configuredLogger.Logger

type GT06Protocol struct {
	LoginInformation *LoginData
	DeviceType       types.DeviceType
}

func (p *GT06Protocol) GetDeviceID() string {
	return p.LoginInformation.TerminalID
}

func (p *GT06Protocol) GetDeviceType() types.DeviceType {
	return p.DeviceType
}

func (p *GT06Protocol) SetDeviceType(t types.DeviceType) {
	p.DeviceType = t
}

func (p *GT06Protocol) GetProtocolType() types.DeviceProtocolType {
	return types.DeviceProtocolType_GT06
}

func (p *GT06Protocol) Login(reader *bufio.Reader) (ack []byte, byteToSkip int, e error) {
	if !p.IsValidHeader(reader) {
		return nil, 0, errs.ErrUnknownProtocol
	}

	// this should have been a gt06 device
	packet, err := p.parsePacket(reader)
	if err != nil {
		logger.Error("failed to parse gt06 packet ", zap.Error(err))
		return nil, 0, err
	}
	if packet.MessageType == MSG_LoginData {
		p.LoginInformation = packet.Information.(*LoginData)

		byteBuffer := bytes.NewBuffer([]byte{})
		err = p.sendResponse(packet, byteBuffer)
		if err != nil {
			logger.Error("failed to parse gt06 packet ", zap.Error(err))
			return nil, 0, err
		}

		return byteBuffer.Bytes(), 0, nil // nothing to skip since the stream is already consumed
	} else {
		return nil, 0, errs.ErrGT06InvalidLoginInfo
	}
}

func (p *GT06Protocol) ConsumeStream(reader *bufio.Reader, writer io.Writer, asyncStore chan types.DeviceStatus) error {
	for {
		packet, err := p.parsePacket(reader)
		if err != nil {
			return err
		}
		err = p.sendResponse(packet, writer)
		if err != nil {
			return err
		}

		protoPacket := packet.ToProtobufDeviceStatus(p.GetDeviceID(), p.DeviceType)
		asyncStore <- *protoPacket
	}
}

func (p *GT06Protocol) sendResponse(parsedPacket *Packet, writer io.Writer) (err error) {
	defer func() {
		if condition := recover(); condition != nil {
			err = condition.(error)
			logger.Error("failed to write response packet", zap.Error(err))
		}
	}()

	responsePacket := ResponsePacket{
		StartBit:                parsedPacket.StartBit,
		PacketLength:            parsedPacket.PacketLength,
		ProtocolNumber:          int8(parsedPacket.MessageType),
		InformationSerialNumber: parsedPacket.InformationSerialNumber,
		Crc:                     parsedPacket.Crc,
		StopBits:                parsedPacket.StopBits,
	}
	_, err = writer.Write(responsePacket.ToBytes())
	checkErr(err)
	return nil
}

func (p *GT06Protocol) parsePacket(reader *bufio.Reader) (packet *Packet, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if err != io.EOF {
				err = errs.ErrGT06BadDataPacket
			}
		}
	}()

	packet = &Packet{}

	// start bit
	checkErr(binary.Read(reader, binary.BigEndian, &packet.StartBit))

	// packet length
	checkErr(binary.Read(reader, binary.BigEndian, &packet.PacketLength))

	// packet data
	packetData := make([]byte, packet.PacketLength-4) // 2 for crc, 2 for serial number
	_, err = io.ReadFull(reader, packetData)
	checkErr(err)

	// packet data to packet
	checkErr(p.parsePacketData(bufio.NewReader(bytes.NewReader(packetData)), packet))

	// information serial number
	checkErr(binary.Read(reader, binary.BigEndian, &packet.InformationSerialNumber))

	// crc
	checkErr(binary.Read(reader, binary.BigEndian, &packet.Crc))

	// stop bits
	checkErr(binary.Read(reader, binary.BigEndian, &packet.StopBits))

	if packet.StopBits != 0x0d0a {
		panic(err)
	}

	// validate crc
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
		logger.Sugar().Errorf("Invalid crc. Excpected %x, got %x", expectedCrc, packet.Crc)
		return nil, errs.ErrBadCrc
	}
	return
}

func (p *GT06Protocol) parsePacketData(reader *bufio.Reader, packet *Packet) error {
	protocolNumByte, err := reader.ReadByte()
	msgType := MessageType(protocolNumByte)
	if msgType == MSG_Invalid {
		logger.Sugar().Errorf("Invalid message type: %x", protocolNumByte)
		remainingData, err := p.consumePacket(reader)
		if err != nil {
			return err
		}
		logger.Sugar().Errorln("Invalid message type: ", hex.Dump(remainingData))
		return errs.ErrGT06BadDataPacket
	}

	packet.MessageType = msgType

	// TODO: parse packetInfoBytes
	packet.Information, err = p.parsePacketInformation(reader, packet.MessageType)
	if err != nil {
		return err
	}

	return nil
}

func (p *GT06Protocol) consumePacket(reader *bufio.Reader) ([]byte, error) {
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

func (p *GT06Protocol) parsePacketInformation(reader *bufio.Reader, messageType MessageType) (interface{}, error) {
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
		return parsedInfo, err
	} else {
		return nil, errs.ErrGT06BadDataPacket
	}
}

func (p *GT06Protocol) parseLoginInformation(reader *bufio.Reader) (interface{}, error) {
	var loginInfo LoginData

	var imeiBytes [8]byte
	err := binary.Read(reader, binary.BigEndian, &imeiBytes)
	if err != nil {
		return nil, errs.ErrGT06InvalidLoginInfo
	}
	loginInfo.TerminalID = hex.EncodeToString(imeiBytes[:])[1:] // imei is 15 chars

	err = binary.Read(reader, binary.BigEndian, &loginInfo.TerminalType)
	if err != nil {
		return nil, errs.ErrGT06InvalidLoginInfo
	}

	var timezoneOffset int16
	err = binary.Read(reader, binary.BigEndian, &timezoneOffset)
	if err != nil {
		return nil, errs.ErrGT06InvalidLoginInfo
	}
	timezonePart := int(timezoneOffset >> 4)
	hours := timezonePart / 100
	minutes := timezonePart % 100
	zoneOffset := (timezoneOffset & 0x0008) >> 3
	if zoneOffset == 1 {
		zoneOffset = -1
	} else {
		zoneOffset = 1
	}
	loginInfo.Timezone = time.FixedZone("", int(zoneOffset)*(hours*60*60+minutes*60))

	return &loginInfo, nil
}

func (p *GT06Protocol) parsePositioningData(reader *bufio.Reader) (positionInfo interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if err != io.EOF {
				err = errs.ErrGT06BadDataPacket
			}
		}
	}()

	var parsed PositioningInformation

	gpsInfo, err := p.parseGPSInformation(reader)
	checkErr(err)
	parsed.GpsInformation = gpsInfo

	lbsInfo, err := p.parseLBSInformation(reader)
	checkErr(err)
	parsed.LBSInfo = lbsInfo

	// ACC
	var b byte
	checkErr(binary.Read(reader, binary.BigEndian, &b))
	parsed.ACCHigh = b == 0x01 // 00 is low, 01 is high

	// data reporting mode
	checkErr(binary.Read(reader, binary.BigEndian, &parsed.DataReportingMode))

	checkErr(binary.Read(reader, binary.BigEndian, &b))
	parsed.GPSRealTime = b == 0x00 // 00 is realtime, 01 is re-upload

	// mileage statistics
	checkErr(binary.Read(reader, binary.BigEndian, &parsed.MileageStatistics))
	return &parsed, nil
}

func (p *GT06Protocol) parseAlarmData(reader *bufio.Reader) (alarmInfo AlarmInformation, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if err != io.EOF {
				err = errs.ErrGT06BadDataPacket
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

func (p *GT06Protocol) parseHeartbeatData(reader *bufio.Reader) (heartbeat HeartbeatData, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if err != io.EOF {
				err = errs.ErrGT06BadDataPacket
			}
		}
	}()

	var b byte
	checkErr(binary.Read(reader, binary.BigEndian, &b))

	heartbeat.TerminalInformation, err = p.parseTerminalInfoFromByte(b)
	checkErr(err)

	checkErr(binary.Read(reader, binary.BigEndian, &b))
	heartbeat.BatteryLevel = BatteryLevel(b)
	if heartbeat.BatteryLevel == VL_Invalid {
		return heartbeat, errs.ErrGT06InvalidVoltageLevel
	}

	checkErr(binary.Read(reader, binary.BigEndian, &b))
	heartbeat.GSMSignalStrength = GSMSignalStrength(b)
	if heartbeat.GSMSignalStrength == GSM_Invalid {
		return heartbeat, errs.ErrGT06InvalidGSMSignalStrength
	}

	checkErr(binary.Read(reader, binary.BigEndian, &heartbeat.ExtendedPortStatus))
	return
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func (p *GT06Protocol) parseGPSInformation(reader *bufio.Reader) (gpsInfo GPSInformation, err error) {
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

	// TODO: parse the 16-bit course to detailed fields
	// course/heading
	var courseValue uint16
	checkErr(binary.Read(reader, binary.BigEndian, &courseValue))
	gpsInfo.Course = p.parseGpsCourse(courseValue)

	return
}

func (p *GT06Protocol) parseGpsCourse(courseValue uint16) (course GPSCourse) {
	b1 := byte(courseValue >> 8)

	course.IsRealtime = b1&0x20 == 0x00     // byte 1, bit 5 is 0
	course.IsDifferential = b1&0x20 == 0x20 // byte 1, bit 5 is 1
	course.Positioned = b1&0x10 == 0x10     // byte 1, bit 4 is 0
	course.Longitude = b1&0x08 == 0x08      // byte 1, bit 3 is 0
	course.Latitude = b1&0x04 == 0x04       // byte 1, bit 2 is 0

	course.Degree = courseValue & 0x03ff // byte 1 (bit 1, 0), byte 2
	return
}

func (p *GT06Protocol) parseTimestamp(reader *bufio.Reader) (timestamp time.Time, err error) {
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

	timestamp = time.Date(int(year), time.Month(month), int(day), int(hour), int(minute), int(second), 0, p.LoginInformation.Timezone)
	return
}

func (p *GT06Protocol) parseLBSInformation(reader *bufio.Reader) (lbsInfo LBSInformation, err error) {
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

func (p *GT06Protocol) parseStatusInformation(reader *bufio.Reader) (statusInfo StatusInformation, err error) {
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

func (p *GT06Protocol) parseTerminalInfoFromByte(terminalInfoByte byte) (TerminalInformation, error) {
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

func (p *GT06Protocol) IsValidHeader(reader *bufio.Reader) bool {
	header, err := reader.Peek(2)
	if err != nil {
		return false
	}

	if bytes.Equal(header, []byte{0x78, 0x78}) || bytes.Equal(header, []byte{0x79, 0x79}) {
		return true
	}
	return false
}
