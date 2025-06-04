package obdii2g

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
	"go.uber.org/zap"
)

var logger = configuredLogger.Logger

type AquilaOBDII2GProtocol struct {
	Imei       string
	DeviceType types.DeviceType
}

const (
	loginEventCode    = "15"
	readTimeout       = 40 * time.Second
	keepAliveInterval = 30 * time.Second
)

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

func (a *AquilaOBDII2GProtocol) ConsumeStream(reader *bufio.Reader, writer io.Writer, store store.Store) error {
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := a.setReadTimeout(writer, readTimeout); err != nil {
				logger.Error("Failed to refresh read deadline", zap.Error(err))
			}
		default:
			packet, err := reader.ReadString('*')
			logger.Sugar().Infoln("Full Packet", packet)
			if err != nil {
				if errors.Is(err, io.EOF) {
					logger.Info("Connection closed gracefully")
					return nil
				}
				return a.handleStreamError(err, reader)
			}

			if err := a.processPacket(packet, store); err != nil {
				logger.Warn("Packet processing failed", zap.Error(err))
				continue
			}
		}
	}
}

func (a *AquilaOBDII2GProtocol) processPacket(raw string, store store.Store) error {
	// Validate and parse packet
	pkt, err := ParsePacket(raw)
	if err != nil {
		return fmt.Errorf("packet validation failed: %w", err)
	}

	// Convert to protobuf format
	status, err := pkt.ToProtobuf()
	if err != nil {
		return fmt.Errorf("proto conversion failed: %w", err)
	}

	// Send to store with timeout
	select {
	case store.GetProcessChan() <- status:
	case <-time.After(100 * time.Millisecond):
		logger.Warn("Dropping packet due to store buffer full")
	}

	return nil
}

func (a *AquilaOBDII2GProtocol) handleStreamError(err error, reader *bufio.Reader) error {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		logger.Warn("Connection timeout, closing")
		return nil
	}

	// Log remaining data for debugging
	if buf := reader.Buffered(); buf > 0 {
		peeked, _ := reader.Peek(min(buf, 256))
		logger.Debug("Remaining buffer content",
			zap.ByteString("data", peeked),
			zap.Int("bytes", buf))
	}

	return fmt.Errorf("stream read error: %w", err)
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
