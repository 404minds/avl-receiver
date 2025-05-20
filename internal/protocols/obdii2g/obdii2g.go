package obdii2g

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"time"

	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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
	// Peek first 2 bytes to verify header
	header, err := reader.Peek(2)
	logger.Sugar().Infoln(header) //  INFO    obdii2g/obdii2g.go:47   [36 36]
	if err != nil {
		return nil, 0, fmt.Errorf("header peek failed: %w", err)
	}
	if !bytes.Equal(header, []byte{0x24, 0x24}) { // $$ in ASCII
		return nil, 0, errs.ErrUnknownProtocol
	}

	peekSize := 40
	peeked, err := reader.Peek(peekSize)
	if err != nil && err != bufio.ErrBufferFull {
		return nil, 0, fmt.Errorf("imei peek failed: %w", err)
	}

	// Find first comma after header ($$CLIENT...)
	firstComma := bytes.IndexByte(peeked[2:], ',') + 2
	if firstComma < 2 {
		return nil, 0, errors.New("invalid packet format - first comma")
	}

	// Find second comma (IMEI end)
	secondComma := bytes.IndexByte(peeked[firstComma+1:], ',') + firstComma + 1
	if secondComma <= firstComma {
		return nil, 0, errors.New("invalid packet format - second comma")
	}

	// Extract IMEI (between first and second commas)
	imeiBytes := peeked[firstComma+1 : secondComma]
	if len(imeiBytes) != 15 { // Validate IMEI length
		return nil, 0, errors.New("invalid IMEI length")
	}

	imei := string(imeiBytes)

	a.Imei = imei

	return []byte{}, 0, nil
}

func (a *AquilaOBDII2GProtocol) ConsumeStream(reader *bufio.Reader, responseWriter io.Writer, dataStore store.Store) error {
	for {
		if err := a.setReadTimeout(responseWriter, 40*time.Second); err != nil {
			logger.Error("Failed to set read timeout", zap.Error(err))
			return err
		}

		packet, err := reader.ReadString('\n')
		logger.Sugar().Infoln("full value", packet)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			logger.Sugar().Info("----------------------------------------------")
			logger.Sugar().Info("Remaining unread data:", reader)
			return errors.Wrap(err, "failed to read packet")
		}

		asyncStore := dataStore.GetProcessChan()

		protoPacket := &types.DeviceStatus{
			Imei:       a.Imei,
			DeviceType: types.DeviceType_AQUILA,
			Timestamp:  timestamppb.Now(),
			Position:   &types.GPSPosition{},
			VehicleStatus: &types.VehicleStatus{
				Ignition: new(bool),
			},
		}
		asyncStore <- protoPacket

	}
}

func (a *AquilaOBDII2GProtocol) SendCommandToDevice(writer io.Writer, command string) error {
	cmd := fmt.Sprintf("#%s\\r\\n", command)
	_, err := writer.Write([]byte(cmd))
	return err
}

func (a *AquilaOBDII2GProtocol) setReadTimeout(writer io.Writer, timeout time.Duration) error {
	if conn, ok := writer.(net.Conn); ok {
		return conn.SetReadDeadline(time.Now().Add(timeout))
	}
	return nil
}
