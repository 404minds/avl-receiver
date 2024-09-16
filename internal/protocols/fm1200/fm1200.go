package fm1200

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"io"
	"log"

	"github.com/404minds/avl-receiver/internal/crc"
	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/types"
)

var logger = configuredLogger.Logger

type FM1200Protocol struct {
	Imei       string
	DeviceType types.DeviceType
}

func (t *FM1200Protocol) GetDeviceID() string {
	return t.Imei
}

func (p *FM1200Protocol) GetDeviceType() types.DeviceType {
	return p.DeviceType
}

func (p *FM1200Protocol) SetDeviceType(t types.DeviceType) {
	p.DeviceType = t
}

func (p *FM1200Protocol) GetProtocolType() types.DeviceProtocolType {
	return types.DeviceProtocolType_FM1200
}

func (t *FM1200Protocol) Login(reader *bufio.Reader) (ack []byte, bytesToSkip int, e error) {
	imei, bytesToSkip, err := t.peekImei(reader)
	if err != nil {
		return nil, bytesToSkip, err
	}
	// TODO: in case of unauthorized device, reply with 0x00 ack
	if !t.isImeiAuthorized(imei) {
		return nil, bytesToSkip, errs.ErrUnauthorizedDevice
	}

	t.Imei = imei // maybe store this in redis if stream consume happens in a different process

	return []byte{0x01}, bytesToSkip, nil
}

func (t *FM1200Protocol) ConsumeStream(reader *bufio.Reader, responseWriter io.Writer, asyncStore chan types.DeviceStatus) error {
	for {
		err := t.consumeMessage(reader, asyncStore, responseWriter)
		if err != nil {
			if err != io.EOF {
				logger.Error("failed to consume message", zap.Error(err))
			}
			return err
		}
	}
}

func (t *FM1200Protocol) consumeMessage(reader *bufio.Reader, asyncStore chan types.DeviceStatus, responseWriter io.Writer) (err error) {
	var headerZeros uint32
	err = binary.Read(reader, binary.BigEndian, &headerZeros)
	if err != nil {
		return err
	}
	if headerZeros != 0x0000 {
		return errors.Wrapf(errs.ErrFM1200BadDataPacket, "error at header Zeroes")
	}

	var dataLen uint32
	err = binary.Read(reader, binary.BigEndian, &dataLen)
	if err != nil {
		return err
	}

	logger.Sugar().Info("consumeMessage Data length: ", dataLen)
	dataBytes := make([]byte, dataLen)
	_, err = io.ReadFull(reader, dataBytes)
	if err != nil {
		return errors.Wrapf(errs.ErrFM1200BadDataPacket, "error at Data length")
	}
	logger.Sugar().Info("consumeMessage Data Byte: ", dataBytes)
	dataReader := bufio.NewReader(bytes.NewReader(dataBytes))

	parsedPacket, err := t.parseDataToRecord(dataReader)
	if err != nil {
		return err
	}

	err = binary.Read(reader, binary.BigEndian, &parsedPacket.CRC)
	if err != nil {
		return errors.Wrapf(errs.ErrFM1200BadDataPacket, "error at parsed Packet CRC")
	}

	valid := t.ValidateCrc(dataBytes, parsedPacket.CRC)
	if !valid {
		return errs.ErrBadCrc
	}

	for _, record := range parsedPacket.Data {
		r := Record{
			Record: record,
			IMEI:   t.Imei,
		}
		protoRecord := r.ToProtobufDeviceStatus()
		asyncStore <- *protoRecord
	}
	logger.Sugar().Infof("stored %d records", len(parsedPacket.Data))

	err = binary.Write(responseWriter, binary.BigEndian, int32(parsedPacket.NumberOfData))
	if err != nil {
		return err
	}
	return nil
}

func (t *FM1200Protocol) parseDataToRecord(reader *bufio.Reader) (*AvlDataPacket, error) {
	var packet AvlDataPacket
	var err error

	// coded id
	packet.CodecID, err = reader.ReadByte()
	if err != nil {
		return nil, err
	}

	logger.Sugar().Info("parseDataToRecord:  codec: ", packet.CodecID)

	// number of data
	packet.NumberOfData, err = reader.ReadByte()
	if err != nil {
		return nil, err
	}

	logger.Sugar().Info("parseDataRecord: NumberofData ", packet.NumberOfData)

	// parse each record
	for i := uint8(0); i < packet.NumberOfData; i++ { //TODO range == packet.NumberOfData currently just for debugging

		logger.Sugar().Info("parseDataToRecord: Data Number: ", i)
		record, err := t.readSingleRecord(reader, packet.CodecID)
		if err != nil {
			return nil, err
		}
		packet.Data = append(packet.Data, *record)
	}

	endNumRecords, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	logger.Sugar().Info("parseDataToRecord endNumRecords: ", endNumRecords)
	if endNumRecords != packet.NumberOfData {
		return nil, errors.Wrapf(errs.ErrFM1200BadDataPacket, "error end Num Records != packet.NumberOfData")
	}
	return &packet, nil
}

func (t *FM1200Protocol) readSingleRecord(reader *bufio.Reader, codecID uint8) (*AvlRecord, error) {
	var record AvlRecord
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

	logger.Sugar().Info("readSingleRecord: Priority: ", record.Priority)

	// gps element
	gpsElement, err := t.parseGpsElement(reader)
	if err != nil {
		return nil, err
	}
	record.GPSElement = gpsElement

	// io elements
	ioElement, err := t.parseIOElements(reader, codecID)
	if err != nil {
		return nil, err
	}
	record.IOElement = *ioElement

	return &record, nil
}

func (t *FM1200Protocol) parseIOElements(reader *bufio.Reader, codecID uint8) (*IOElement, error) {
	ioElement := &IOElement{}

	// EventID
	if codecID == 0x8E {
		err := binary.Read(reader, binary.BigEndian, &ioElement.EventID)
		if err != nil {
			return nil, err
		}
		logger.Sugar().Info("parseIOElements: eventID: ", ioElement.EventID)
	} else {
		eventID, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		ioElement.EventID = uint16(eventID)
		logger.Sugar().Info("parseIOElements: eventID: ", ioElement.EventID)
	}

	// Number of properties
	if codecID == 0x8E {
		err := binary.Read(reader, binary.BigEndian, &ioElement.NumProperties)
		if err != nil {
			return nil, err
		}
	} else {
		numOfProperties, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		ioElement.NumProperties = uint16(numOfProperties)
	}

	var err1, err2, err3, err4, err5 error
	ioElement.Properties1B, err1 = t.read1BProperties(reader, codecID)
	logger.Sugar().Info("parseIOElements: properties1B error: ", err1)

	ioElement.Properties2B, err2 = t.read2BProperties(reader, codecID)
	logger.Sugar().Info("parseIOElements: properties2B error: ", err2)

	ioElement.Properties4B, err3 = t.read4BProperties(reader, codecID)
	logger.Sugar().Info("parseIOElements: properties4B error: ", err3)

	ioElement.Properties8B, err4 = t.read8BProperties(reader, codecID)
	logger.Sugar().Info("parseIOElements: properties8B error: ", err4)

	if codecID == 0x8E {
		ioElement.PropertiesNXB, err5 = t.readNXBProperties(reader, codecID)
		logger.Sugar().Info("parseIOElements: propertiesNXB: ", ioElement.PropertiesNXB)
		logger.Sugar().Info("parseIOElements: propertiesNXB error: ", err5)
	}
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
		return nil, errors.Wrapf(errs.ErrFM1200BadDataPacket, "error at IO elements")
	}

	return ioElement, nil
}

func (t *FM1200Protocol) read1BProperties(reader *bufio.Reader, codecID uint8) (map[IOProperty]uint8, error) {
	propertyMap, err := t.readNByteProperties(1, reader, codecID)
	if err != nil {
		return nil, err
	}

	properties := make(map[IOProperty]uint8)
	for k, v := range propertyMap {
		properties[k] = v.(uint8)
	}

	return properties, nil
}

func (t *FM1200Protocol) read2BProperties(reader *bufio.Reader, codecID uint8) (map[IOProperty]uint16, error) {
	propertyMap, err := t.readNByteProperties(2, reader, codecID)
	if err != nil {
		return nil, err
	}

	properties := make(map[IOProperty]uint16)
	for k, v := range propertyMap {
		properties[k] = v.(uint16)
	}

	return properties, nil
}

func (t *FM1200Protocol) read4BProperties(reader *bufio.Reader, codecID uint8) (map[IOProperty]uint32, error) {
	propertyMap, err := t.readNByteProperties(4, reader, codecID)
	logger.Sugar().Info("read4BProperties: error ", err)

	if err != nil {
		return nil, err
	}

	properties := make(map[IOProperty]uint32)
	for k, v := range propertyMap {
		properties[k] = v.(uint32)
	}

	return properties, nil
}

func (t *FM1200Protocol) read8BProperties(reader *bufio.Reader, codecID uint8) (map[IOProperty]uint64, error) {
	propertyMap, err := t.readNByteProperties(8, reader, codecID)
	logger.Sugar().Info("read8BProperties: error ", err)
	if err != nil {
		return nil, err
	}

	properties := make(map[IOProperty]uint64)
	for k, v := range propertyMap {
		properties[k] = v.(uint64)
	}

	return properties, nil
}

func (t *FM1200Protocol) readNXBProperties(reader *bufio.Reader, codecID uint8) (map[IOProperty][]byte, error) {
	propertyMap, err := t.readNXByteProperties(reader, codecID)
	if err != nil {
		log.Println("readNXBProperties: error ", err)
		return nil, err
	}

	properties := make(map[IOProperty][]byte)
	for k, v := range propertyMap {
		byteValue, ok := v.([]byte)
		if !ok {
			return nil, fmt.Errorf("unexpected type for property %d: expected []byte, got %T", k, v)
		}
		properties[k] = byteValue
	}

	return properties, nil
}

func (t *FM1200Protocol) readNByteProperties(n int, reader *bufio.Reader, codecID uint8) (map[IOProperty]interface{}, error) {
	var numProperties uint16
	if codecID == 0x8E {
		err := binary.Read(reader, binary.BigEndian, &numProperties)
		if err != nil {
			logger.Sugar().Info("readNByteProperties: error:  ", err)
			return nil, err
		}
	} else {
		var numProperties8 uint8
		err := binary.Read(reader, binary.BigEndian, &numProperties8)
		if err != nil {
			logger.Sugar().Info("readNByteProperties: error:  ", err)
			return nil, err
		}
		numProperties = uint16(numProperties8)
	}

	properties := make(map[IOProperty]interface{})
	for i := uint16(0); i < numProperties; i++ {
		var propertyID uint16
		if codecID == 0x8E {
			err := binary.Read(reader, binary.BigEndian, &propertyID)
			if err != nil {
				logger.Sugar().Info("readNByteProperties: error:  ", err)
				return nil, err
			}
		} else {
			var propertyID8 uint8
			err := binary.Read(reader, binary.BigEndian, &propertyID8)
			if err != nil {
				logger.Sugar().Info("readNByteProperties: error:  ", err)
				return nil, err
			}
			propertyID = uint16(propertyID8)
		}

		property := IOProperty(propertyID)

		propBytes := make([]byte, n)
		err := binary.Read(reader, binary.BigEndian, &propBytes)
		if err != nil {
			logger.Sugar().Info("readNByteProperties: error:  ", err)
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

func (t *FM1200Protocol) readNXByteProperties(reader *bufio.Reader, codecID uint8) (map[IOProperty]interface{}, error) {
	var numProperties uint16
	if codecID == 0x8E {
		err := binary.Read(reader, binary.BigEndian, &numProperties)
		if err != nil {
			logger.Sugar().Info("readNXByteProperties: error:  ", err)
			return nil, err
		}
	} else {
		var numProperties8 uint8
		err := binary.Read(reader, binary.BigEndian, &numProperties8)
		if err != nil {
			logger.Sugar().Info("readNXByteProperties: error:  ", err)
			return nil, err
		}
		numProperties = uint16(numProperties8)
	}

	properties := make(map[IOProperty]interface{})
	for i := uint16(0); i < numProperties; i++ {
		var propertyID uint16
		if codecID == 0x8E {
			err := binary.Read(reader, binary.BigEndian, &propertyID)
			if err != nil {
				logger.Sugar().Info("readNXByteProperties: error:  ", err)
				return nil, err
			}
		} else {
			var propertyID8 uint8
			err := binary.Read(reader, binary.BigEndian, &propertyID8)
			if err != nil {
				logger.Sugar().Info("readNXByteProperties: error:  ", err)
				return nil, err
			}
			propertyID = uint16(propertyID8)
		}

		property := IOProperty(propertyID)

		var valueLen uint8
		err := binary.Read(reader, binary.BigEndian, &valueLen)
		if err != nil {
			logger.Sugar().Info("readNXByteProperties: error:  ", err)
			return nil, err
		}

		propBytes := make([]byte, valueLen)
		_, err = io.ReadFull(reader, propBytes)
		if err != nil {
			logger.Sugar().Info("readNXByteProperties: error:  ", err)
			return nil, err
		}

		properties[property] = propBytes
	}

	return properties, nil
}

func (t *FM1200Protocol) parseGpsElement(reader *bufio.Reader) (gpsElement GpsElement, err error) {
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

func (t *FM1200Protocol) peekImei(reader *bufio.Reader) (imei string, bytesConsumed int, e error) {
	imeiLenBytes, err := reader.Peek(2)
	if err != nil {
		return "", 0, err
	}

	imeiLen := binary.BigEndian.Uint16(imeiLenBytes)
	if imeiLen != 15 {
		return "", 0, errs.ErrUnknownProtocol
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

func (t *FM1200Protocol) isImeiAuthorized(imei string) bool {
	logger.Sugar().Infof("IMEI %s is authorized", imei)
	return true
}

func (t *FM1200Protocol) ValidateCrc(data []byte, expectedCrc uint32) bool {
	calculatedCrc := crc.CrcTeltonika(data)
	return uint32(calculatedCrc) == expectedCrc
}
