package howen

import (
	"bufio"
	"encoding/json"
	"fmt"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"io"
)

var logger = configuredLogger.Logger

type HOWENWS struct {
	DeviceType types.DeviceType
}

func (p *HOWENWS) GetDeviceID() string {
	return ""
}

func (p *HOWENWS) GetDeviceType() types.DeviceType {
	return p.DeviceType
}

func (p *HOWENWS) SetDeviceType(t types.DeviceType) {
	p.DeviceType = t
}

func (p *HOWENWS) GetProtocolType() types.DeviceProtocolType {
	return types.DeviceProtocolType_HOWENWS
}

func (p *HOWENWS) Login(reader *bufio.Reader) (ack []byte, byteToSkip int, e error) {
	return nil, 0, nil
}

func (p *HOWENWS) ConsumeStream(reader *bufio.Reader, writer io.Writer, dataStore store.Store) error {
	return nil
}

// send command to device
func (p *HOWENWS) SendCommandToDevice(writer io.Writer, command string) error {
	// Command in HEX for "getinfo"
	return nil
}

func (p *HOWENWS) ConsumeConnection(conn *websocket.Conn, dataStore store.Store) error {
	logger.Sugar().Info("consume connection called")
	for {
		if conn == nil {
			logger.Sugar().Error("Connection is nil, stopping consumption.")
			return errors.New("connection is nil")
		}

		err := p.ConsumeMessage(conn, dataStore)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				logger.Sugar().Error("WebSocket closed unexpectedly:", err)
			} else {
				logger.Sugar().Info("WebSocket read error:", err)
			}
			return err
		}
	}
}

func (p *HOWENWS) ConsumeMessage(conn *websocket.Conn, dataStore store.Store) error {
	// Read message from WebSocket
	_, message, err := conn.ReadMessage()
	if err != nil {
		logger.Sugar().Error("Error reading WebSocket message:", err)
		return errors.Wrap(err, "error reading WebSocket message")
	}

	logger.Sugar().Info("Received WebSocket message:", string(message))

	// Unmarshal the message to check the action type
	var actionData ActionData
	if err := json.Unmarshal(message, &actionData); err != nil {
		return errors.Wrap(err, "error unmarshaling action data")
	}

	asyncStore := dataStore.GetProcessChan()

	switch actionData.Action {
	case "80003":
		gpsPacket, err := p.parseGPSPacket(message)
		if err != nil {
			return errors.Wrap(err, "error parsing GPS packet")
		}
		protoReply := gpsPacket.ToProtobufDeviceStatusGPS()
		asyncStore <- protoReply
	case "80004":
		alarmPacket, err := p.parseAlarmMessage(message)
		if err != nil {
			return errors.Wrap(err, "error parsing Alarm packet")
		}
		protoReply := alarmPacket.ToProtobufDeviceStatusAlarm()
		asyncStore <- protoReply
	default:
		logger.Sugar().Infof("Unhandled action type: %s", actionData.Action)
	}

	return nil
}

// parse response of type 80003 (gps)
func (p *HOWENWS) parseGPSPacket(jsonData []byte) (*GPSPacket, error) {
	var gpsPacket GPSPacket
	err := json.Unmarshal(jsonData, &gpsPacket)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}
	return &gpsPacket, nil
}

// parse response of type 80005 (offline/online)
func (p *HOWENWS) parseDeviceStatus(jsonData []byte) (*DeviceStatus, error) {
	var deviceStatus DeviceStatus
	err := json.Unmarshal(jsonData, &deviceStatus)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}
	return &deviceStatus, nil
}

func (p *HOWENWS) parseAlarmMessage(jsonData []byte) (*AlarmMessage, error) {
	var alarmMessage AlarmMessage
	err := json.Unmarshal(jsonData, &alarmMessage)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}
	return &alarmMessage, nil
}
