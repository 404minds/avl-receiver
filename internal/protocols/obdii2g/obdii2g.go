package obdii2g

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var logger = configuredLogger.Logger

type AquilaOBDII2GProtocol struct {
	Imei       string
	DeviceType types.DeviceType
}

func (a *AquilaOBDII2GProtocol) GetDeviceID() string {
	return a.Imei
}

func (a *AquilaOBDII2GProtocol) GetDeviceType() types.DeviceType {
	return a.DeviceType
}

func (a *AquilaOBDII2GProtocol) SetDeviceType(dt types.DeviceType) {
	a.DeviceType = dt
}

func (a *AquilaOBDII2GProtocol) GetProtocolType() types.DeviceProtocolType {
	return types.DeviceProtocolType_OBDII2G
}

func (a *AquilaOBDII2GProtocol) Login(reader *bufio.Reader) ([]byte, int, error) {
	peeked, err := reader.Peek(32)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to peek login packet")
	}

	packetStr := string(peeked)
	logger.Sugar().Infoln("reader", peeked)
	logger.Sugar().Infoln("string reader", packetStr)

	// Verify packet starts with $$ header
	if !strings.HasPrefix(packetStr, "$$") {
		return nil, 0, errs.ErrUnknownProtocol
	}

	parts := strings.Split(packetStr, ",")
	if len(parts) < 3 {
		return nil, 0, errors.New("invalid login packet format")
	}

	// Extract IMEI from second field (index 1)
	imei := parts[1]
	if !a.isImeiAuthorized(imei) {
		return nil, len(peeked), errs.ErrUnauthorizedDevice
	}

	a.Imei = imei

	// Proper ACK format based on protocol document example
	ack := []byte("*" + hex.EncodeToString([]byte{calculateChecksum(packetStr)}))
	return ack, len(peeked), nil
}

func (a *AquilaOBDII2GProtocol) ConsumeStream(reader *bufio.Reader, writer io.Writer, store store.Store) error {
	for {
		packet, err := reader.ReadString('*')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.Wrap(err, "failed to read packet")
		}

		// Validate checksum
		parts := strings.Split(packet, "*")
		if len(parts) != 2 {
			return errors.New("invalid packet format")
		}

		calculatedChecksum := calculateChecksum(parts[0])
		receivedChecksum, err := hex.DecodeString(parts[1])
		if err != nil || calculatedChecksum != receivedChecksum[0] {
			return errors.New("invalid checksum")
		}

		// Parse packet
		status, err := a.parsePacket(parts[0])
		if err != nil {
			return errors.Wrap(err, "failed to parse packet")
		}

		// Send to store
		store.GetProcessChan() <- status
	}
}

func (a *AquilaOBDII2GProtocol) parsePacket(packet string) (*types.DeviceStatus, error) {
	parts := strings.Split(packet, ",")
	if len(parts) < 20 {
		return nil, errors.New("invalid packet length")
	}

	logger.Sugar().Infoln("parts obd2  ", parts)
	status := &types.DeviceStatus{
		Imei:       a.Imei,
		DeviceType: types.DeviceType_AQUILA,
		Timestamp:  timestamppb.Now(),
		Position:   &types.GPSPosition{},
		VehicleStatus: &types.VehicleStatus{
			Ignition: new(bool),
		},
	}

	return status, nil
}

func (a *AquilaOBDII2GProtocol) parseEventFlags(vs *types.VehicleStatus, flags uint32) {
	// Implementation based on event flag table
	vs.OverSpeeding = (flags>>2)&1 == 1
	vs.CrashDetection = (flags>>24)&1 == 1
	vs.Towing = (flags>>12)&1 == 1
	vs.UnplugBattery = (flags>>5)&1 == 1
	// Add more flags as needed
}

func (a *AquilaOBDII2GProtocol) parseOBDData(status *types.DeviceStatus, obdData []string) {
	for _, data := range obdData {
		if strings.HasPrefix(data, "010D:") {
			// RPM data example: 010D:06410D15000000
			rpmVal, _ := strconv.ParseInt(data[7:11], 16, 32)
			status.Rpm = int32(rpmVal)
		} else if strings.HasPrefix(data, "51|") {
			// VIN data
			status.Vin = strings.Split(data, "|")[1]
		} else if strings.HasPrefix(data, "52|") {
			// DTC data
			codes := strings.Split(data, "|")
			status.VehicleStatus.DriverDistraction = len(codes) > 1 // Example mapping
		}
		// Add more PID parsers
	}
}

func (a *AquilaOBDII2GProtocol) SendCommandToDevice(writer io.Writer, command string) error {
	cmd := fmt.Sprintf("#%s\\r\\n", command)
	_, err := writer.Write([]byte(cmd))
	return err
}

func (a *AquilaOBDII2GProtocol) isImeiAuthorized(imei string) bool {
	// Implement authorization logic
	return true
}

func calculateChecksum(data string) byte {
	var checksum byte
	for _, c := range []byte(data) {
		checksum ^= c
	}
	return checksum
}

func parseDateTime(dt string) (time.Time, error) {
	if len(dt) != 12 {
		return time.Time{}, errors.New("invalid datetime format")
	}
	return time.Parse("060102150405", dt)
}
