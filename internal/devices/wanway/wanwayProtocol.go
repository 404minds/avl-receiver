package wanway

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"time"

	"github.com/404minds/avl-receiver/internal/crc"
	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
)

var logger = configuredLogger.Logger

type WanwayProtocol struct {
	LoginInformation *WanwayLoginInformation
}

func (p *WanwayProtocol) GetDeviceIdentifier() string {
	return p.LoginInformation.TerminalID
}

func (p *WanwayProtocol) Login(reader *bufio.Reader) (ack []byte, byteToSkip int, e error) {
	if !p.IsWanwayHeader(reader) {
		return nil, 0, errs.ErrNotWanwayDevice
	}

	// this should have been a wanway device
	packet, err := p.parseWanwayPacket(reader)
	if err != nil {
		logger.Sugar().Error("failed to parse wanway packet ", err)
		return nil, 0, err
	}
	if packet.MessageType == MSG_LoginInformation {
		p.LoginInformation = packet.Information.(*WanwayLoginInformation)

		var byteBuffer bytes.Buffer
		var writer = bufio.NewWriter(&byteBuffer)
		err = p.sendResponse(packet, writer)
		if err != nil {
			logger.Sugar().Error("failed to parse wanway packet ", err)
			return nil, 0, err
		}

		return byteBuffer.Bytes(), 0, nil
	} else {
		return nil, 0, errs.ErrWanwayInvalidLoginInfo
	}
}

func (p *WanwayProtocol) ConsumeStream(reader *bufio.Reader, writer *bufio.Writer, storeProcessChan chan interface{}) error {
	for {
		packet, err := p.parseWanwayPacket(reader)
		if err != nil {
			return err
		}
		err = p.sendResponse(packet, writer)
		if err != nil {
			return err
		}
	}
}

func (p *WanwayProtocol) sendResponse(parsedPacket *WanwayPacket, writer *bufio.Writer) (err error) {
	defer func() {
		if condition := recover(); condition != nil {
			err = condition.(error)
			logger.Sugar().Error("failed to write response packet", err)
		}
	}()

	if parsedPacket.MessageType == MSG_LoginInformation {
		responsePacket := ResponsePacket{
			StartBit:                parsedPacket.StartBit,
			PacketLength:            parsedPacket.PacketLength,
			ProtocolNumber:          int8(parsedPacket.MessageType),
			InformationSerialNumber: parsedPacket.InformationSerialNumber,
			Crc:                     parsedPacket.Crc,
			StopBits:                parsedPacket.StopBits,
		}
		_, err := writer.Write(responsePacket.ToBytes())
		checkErr(err)
		checkErr(writer.Flush())
	} else {
		return nil
	}
	return nil
}

func (p *WanwayProtocol) parseWanwayPacket(reader *bufio.Reader) (packet *WanwayPacket, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if err != io.EOF {
				err = errs.ErrWanwayInvalidPacket
			}
		}
	}()

	packet = &WanwayPacket{}

	// start bit
	checkErr(binary.Read(reader, binary.BigEndian, &packet.StartBit))

	// packet length
	checkErr(binary.Read(reader, binary.BigEndian, &packet.PacketLength))

	// packet data
	packetData := make([]byte, packet.PacketLength-2) // 2 for crc
	_, err = io.ReadFull(reader, packetData)
	checkErr(err)

	// packet data to packet
	checkErr(p.parsePacketData(bufio.NewReader(bytes.NewReader(packetData)), packet))

	// crc
	checkErr(binary.Read(reader, binary.BigEndian, &packet.Crc))

	// stop bits
	checkErr(binary.Read(reader, binary.BigEndian, &packet.StopBits))

	if packet.StopBits != 0x0d0a {
		panic(err)
	}

	// validate crc
	expectedCrc := crc.Crc_Wanway(append([]byte{byte(packet.PacketLength)}, packetData...))
	if expectedCrc != packet.Crc {
		// TODO: fix issues with crc validation
		return
		// return nil, errs.ErrWanwayBadCrc
	}
	return
}

func (p *WanwayProtocol) parsePacketData(reader *bufio.Reader, packet *WanwayPacket) error {
	protocolNumByte, err := reader.ReadByte()
	msgType := WanwayMessageTypeFromId(protocolNumByte)
	if msgType == nil {
		return errs.ErrWanwayInvalidPacket
	}
	packet.MessageType = *msgType

	packetInfoBytes := make([]byte, packet.PacketLength-5) // 2 for info serial number, 2 for crc, 1 for msgType
	bytesRead, err := io.ReadFull(reader, packetInfoBytes)
	if bytesRead != int(packet.PacketLength)-5 {
		return errs.ErrWanwayInvalidPacket
	}

	// TODO: parse packetInfoBytes
	packet.Information, err = p.parsePacketInformation(bufio.NewReader(bytes.NewReader(packetInfoBytes)), packet.MessageType)
	if err != nil {
		return err
	}

	err = binary.Read(reader, binary.BigEndian, &packet.InformationSerialNumber)
	if err != nil {
		return errs.ErrWanwayInvalidPacket
	}

	return nil
}

func (p *WanwayProtocol) parsePacketInformation(reader *bufio.Reader, messageType WanwayMessageType) (interface{}, error) {
	if messageType == MSG_LoginInformation {
		parsedInfo, err := p.parseLoginInformation(reader)
		return parsedInfo, err
	} else if messageType == MSG_PositioningData {
		parsedInfo, err := p.parsePositioningData(reader)
		return parsedInfo, err
	} else {
		return nil, errs.ErrWanwayInvalidPacket
	}
}

func (p *WanwayProtocol) parseLoginInformation(reader *bufio.Reader) (interface{}, error) {
	var loginInfo WanwayLoginInformation

	var imeiBytes [8]byte
	err := binary.Read(reader, binary.BigEndian, &imeiBytes)
	if err != nil {
		return nil, errs.ErrWanwayInvalidLoginInfo
	}
	loginInfo.TerminalID = hex.EncodeToString(imeiBytes[:])[1:] // imei is 15 chars

	err = binary.Read(reader, binary.BigEndian, &loginInfo.TerminalType)
	if err != nil {
		return nil, errs.ErrWanwayInvalidLoginInfo
	}

	var timezoneOffset int16
	err = binary.Read(reader, binary.BigEndian, &timezoneOffset)
	if err != nil {
		return nil, errs.ErrWanwayInvalidLoginInfo
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

func (p *WanwayProtocol) parsePositioningData(reader *bufio.Reader) (positionInfo interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			if err != io.EOF {
				err = errs.ErrWanwayInvalidPacket
			}
		}
	}()

	var parsed WanwayPositioningInformation

	gpsInfo, err := p.parseGPSInformation(reader)
	checkErr(err)
	parsed.GPSInfo = gpsInfo

	lbsInfo, err := p.parseLBSInformation(reader)
	checkErr(err)
	parsed.LBSInfo = lbsInfo

	// ACC
	checkErr(binary.Read(reader, binary.BigEndian, &parsed.ACC))

	// data reporting mode
	checkErr(binary.Read(reader, binary.BigEndian, &parsed.DataReportingMode))

	// mileage statistics
	checkErr(binary.Read(reader, binary.BigEndian, &parsed.MileageStatistics))
	return &parsed, nil
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func (p *WanwayProtocol) parseGPSInformation(reader *bufio.Reader) (gpsInfo WanwayGPSInformation, err error) {
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

	gpsInfo.Timestamp = time.Date(int(year), time.Month(month), int(day), int(hour), int(minute), int(second), 0, p.LoginInformation.Timezone)

	x, err := reader.ReadByte()
	checkErr(err)
	gpsInfo.GPSInfoLength = x >> 4
	gpsInfo.NumberOfSatellites = x & 0x0f

	return
}

func (p *WanwayProtocol) parseLBSInformation(reader *bufio.Reader) (lbsInfo WanwayLBSInformation, err error) {
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

func (p *WanwayProtocol) IsWanwayHeader(reader *bufio.Reader) bool {
	header, err := reader.Peek(2)
	if err != nil {
		return false
	}

	if bytes.Equal(header, []byte{0x78, 0x78}) || bytes.Equal(header, []byte{0x79, 0x79}) {
		return true
	}
	return false
}
