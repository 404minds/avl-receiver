package teltonika

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/404minds/avl-receiver/internal/crc"
	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
)

var logger = configuredLogger.Logger

type TeltonikaProtocol struct {
	Imei string
}

func (t *TeltonikaProtocol) GetDeviceIdentifier() string {
	return t.Imei
}

func (t *TeltonikaProtocol) Login(reader *bufio.Reader) (ack []byte, bytesToSkip int, e error) {
	imei, bytesToSkip, err := t.peekImei(reader)
	if err != nil {
		return nil, bytesToSkip, err
	}
	if !t.isImeiAuthorized(imei) {
		return nil, bytesToSkip, errs.ErrTeltonikaUnauthorizedDevice
	}

	t.Imei = imei // maybe store this in redis if stream consume happens in a different process

	return []byte{0x01}, bytesToSkip, nil
}

func (t *TeltonikaProtocol) ConsumeStream(reader *bufio.Reader, writer *bufio.Writer, storeProcessChan chan interface{}) error {
	for {
		err := t.consumeMessage(reader, storeProcessChan, writer)
		if err != nil {
			if err != io.EOF {
				logger.Sugar().Error("failed to consume message", err)
				return fmt.Errorf("failed to consume message %w", err)
			}
			return err
		}
	}
}

func (t *TeltonikaProtocol) consumeMessage(reader *bufio.Reader, storeProcessChan chan interface{}, writer *bufio.Writer) (err error) {
	var headerZeros uint32
	err = binary.Read(reader, binary.BigEndian, &headerZeros)
	if err != nil {
		return err
	}
	if headerZeros != 0x0000 {
		return errs.ErrTeltonikaInvalidDataPacket
	}

	var dataLen uint32
	err = binary.Read(reader, binary.BigEndian, &dataLen)
	if err != nil {
		return err
	}

	dataBytes := make([]byte, dataLen)
	_, err = io.ReadFull(reader, dataBytes)
	if err != nil {
		return fmt.Errorf("failed to read data bytes: %w", err)
	}
	dataReader := bufio.NewReader(bytes.NewReader(dataBytes))

	parsedPacket, err := t.parseDataToRecord(dataReader)
	if err != nil {
		return err
	}

	err = binary.Read(reader, binary.BigEndian, &parsedPacket.CRC)
	if err != nil {
		return fmt.Errorf("failed to read crc: %w", err)
	}

	valid := t.ValidateCrc(dataBytes, parsedPacket.CRC)
	if !valid {
		return errs.ErrTeltonikaBadCrc
	}

	for _, record := range parsedPacket.Data {
		r := TeltonikaRecord{
			Record: record,
			IMEI:   t.Imei,
		}
		storeProcessChan <- r
	}
	logger.Sugar().Infof("stored %d records", len(parsedPacket.Data))

	err = binary.Write(writer, binary.BigEndian, int32(len(parsedPacket.Data)))
	if err != nil {
		logger.Sugar().Error("failed to write ack for incoming data", err)
		return fmt.Errorf("failed to write ack for incoming data: %w", err)
	}
	err = writer.Flush()
	if err != nil {
		logger.Sugar().Error("failed to flush ack for incoming data", err)
		return fmt.Errorf("failed to flush ack for incoming data: %w", err)
	}
	return nil
}

func (t *TeltonikaProtocol) parseDataToRecord(reader *bufio.Reader) (*TeltonikaAvlDataPacket, error) {
	var packet TeltonikaAvlDataPacket

	// coded id
	codedIdBytes, err := reader.ReadByte()
	packet.CodecID = uint8(codedIdBytes)
	if err != nil {
		return nil, err
	}

	// number of data
	err = binary.Read(reader, binary.BigEndian, &packet.NumberOfData)
	if err != nil {
		return nil, err
	}

	// parse each record
	for i := uint8(0); i < packet.NumberOfData; i++ {
		record, err := t.readSingleRecord(reader)
		if err != nil {
			return nil, err
		}
		packet.Data = append(packet.Data, *record)
	}

	endNumRecords, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	if endNumRecords != packet.NumberOfData {
		return nil, errs.ErrTeltonikaInvalidDataPacket
	}
	return &packet, nil
}

func (t *TeltonikaProtocol) readSingleRecord(reader *bufio.Reader) (*TeltonikaAvlRecord, error) {
	var record TeltonikaAvlRecord
	var err error

	// timestamp
	err = binary.Read(reader, binary.BigEndian, &record.Timestamp)
	if err != nil {
		return nil, err
	}

	// priority
	err = binary.Read(reader, binary.BigEndian, &record.Priority)
	if err != nil {
		return nil, err
	}

	// gps element
	gpsElement, err := t.parseGpsElement(reader)
	if err != nil {
		return nil, err
	}
	record.GPSElement = gpsElement

	// io elements
	ioElement, err := t.parseIOElements(reader)
	if err != nil {
		return nil, err
	}
	record.IOElement = *ioElement

	return &record, nil
}

func (t *TeltonikaProtocol) parseIOElements(reader *bufio.Reader) (ioElement *TeltonikaIOElement, err error) {
	ioElement = &TeltonikaIOElement{}

	// eventId
	err = binary.Read(reader, binary.BigEndian, &ioElement.EventID)
	if err != nil {
		return nil, err
	}

	// numProperties
	err = binary.Read(reader, binary.BigEndian, &ioElement.NumProperties)
	if err != nil {
		return nil, err
	}

	var err1, err2, err3, err4 error
	ioElement.Properties1B, err1 = t.read1BProperties(reader)
	ioElement.Properties2B, err2 = t.read2BProperties(reader)
	ioElement.Properties4B, err3 = t.read4BProperties(reader)
	ioElement.Properties8B, err4 = t.read8BProperties(reader)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return nil, errs.ErrTeltonikaInvalidDataPacket
	}

	return
}

func (t *TeltonikaProtocol) read1BProperties(reader *bufio.Reader) (map[TeltonikaIOProperty]uint8, error) {
	propertyMap, err := t.readNByteProperties(1, reader)
	if err != nil {
		return nil, err
	}

	properties := make(map[TeltonikaIOProperty]uint8)
	for k, v := range propertyMap {
		properties[k] = v.(uint8)
	}

	return properties, nil
}

func (t *TeltonikaProtocol) read2BProperties(reader *bufio.Reader) (map[TeltonikaIOProperty]uint16, error) {
	propertyMap, err := t.readNByteProperties(2, reader)
	if err != nil {
		return nil, err
	}

	properties := make(map[TeltonikaIOProperty]uint16)
	for k, v := range propertyMap {
		properties[k] = v.(uint16)
	}

	return properties, nil
}

func (t *TeltonikaProtocol) read4BProperties(reader *bufio.Reader) (map[TeltonikaIOProperty]uint32, error) {
	propertyMap, err := t.readNByteProperties(4, reader)
	if err != nil {
		return nil, err
	}

	properties := make(map[TeltonikaIOProperty]uint32)
	for k, v := range propertyMap {
		properties[k] = v.(uint32)
	}

	return properties, nil
}

func (t *TeltonikaProtocol) read8BProperties(reader *bufio.Reader) (map[TeltonikaIOProperty]uint64, error) {
	propertyMap, err := t.readNByteProperties(8, reader)
	if err != nil {
		return nil, err
	}

	properties := make(map[TeltonikaIOProperty]uint64)
	for k, v := range propertyMap {
		properties[k] = v.(uint64)
	}

	return properties, nil
}

func (t *TeltonikaProtocol) readNByteProperties(n int, reader *bufio.Reader) (map[TeltonikaIOProperty]interface{}, error) {
	var numProperties uint8
	err := binary.Read(reader, binary.BigEndian, &numProperties)
	if err != nil {
		return nil, err
	}

	properties := make(map[TeltonikaIOProperty]interface{})
	for i := uint8(0); i < numProperties; i++ {
		propertyID, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		property := *IOPropertyFromID(propertyID)

		propBytes := make([]byte, n)
		err = binary.Read(reader, binary.BigEndian, &propBytes)
		if err != nil {
			return nil, err
		}
		if n == 1 {
			properties[property] = uint8(propBytes[0])
		} else if n == 2 {
			properties[property] = binary.BigEndian.Uint16(propBytes)
		} else if n == 4 {
			properties[property] = binary.BigEndian.Uint32(propBytes)
		} else if n == 8 {
			properties[property] = binary.BigEndian.Uint64(propBytes)
		}
	}

	return properties, nil
}

func (t *TeltonikaProtocol) parseGpsElement(reader *bufio.Reader) (gpsElement TeltonikaGpsElement, err error) {
	// longitude
	var i32 uint32
	err = binary.Read(reader, binary.BigEndian, &i32)
	if err != nil {
		return
	}
	gpsElement.Longitude = float32(i32) / 10000000

	// latitude
	err = binary.Read(reader, binary.BigEndian, &i32)
	if err != nil {
		return
	}
	gpsElement.Latitude = float32(i32) / 10000000

	// altitude
	err = binary.Read(reader, binary.BigEndian, &gpsElement.Altitude)
	if err != nil {
		return
	}

	// angle
	err = binary.Read(reader, binary.BigEndian, &gpsElement.Angle)
	if err != nil {
		return
	}

	// satellites
	err = binary.Read(reader, binary.BigEndian, &gpsElement.Satellites)
	if err != nil {
		return
	}

	// speed
	err = binary.Read(reader, binary.BigEndian, &gpsElement.Speed)
	if err != nil {
		return
	}

	return
}

func (t *TeltonikaProtocol) peekImei(reader *bufio.Reader) (imei string, bytesConsumed int, e error) {
	imeiLenBytes, err := reader.Peek(2)
	if err != nil {
		return "", 0, err
	}

	imeiLen := binary.BigEndian.Uint16(imeiLenBytes)
	if imeiLen != 15 {
		return "", 0, errs.ErrNotTeltonikaDevice
	}
	err = binary.Read(bytes.NewReader(imeiLenBytes), binary.BigEndian, &imeiLen)
	if err != nil {
		return "", 0, err
	}

	imeiBytes, err := reader.Peek(int(imeiLen) + 2) // skip imei
	if err != nil {
		return "", 0, err
	}

	imei = string(imeiBytes[2:])

	return imei, 2 + int(imeiLen), nil
}

func (t *TeltonikaProtocol) isImeiAuthorized(imei string) bool {
	logger.Sugar().Infof("IMEI %s is authorized", imei)
	return true
}

func (t *TeltonikaProtocol) ValidateCrc(data []byte, expectedCrc uint32) bool {
	calculatedCrc := crc.Crc_Teltonika(data)
	return uint32(calculatedCrc) == expectedCrc
}
