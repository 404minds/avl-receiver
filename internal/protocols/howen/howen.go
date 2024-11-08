package howen

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
	"github.com/gorilla/websocket"
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

	err := p.ConsumeMessage(conn, dataStore)
	if err != nil {
		return err
	}

	return nil
}

func (p *HOWENWS) ConsumeMessage(conn *websocket.Conn, dataStore store.Store) error {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return errors.Wrap(err, "error reading WebSocket message")
		}

		// Unmarshal the message to check the action type
		var actionData ActionData
		if err := json.Unmarshal(message, &actionData); err != nil {
			return errors.Wrap(err, "error unmarshaling action data")
		}

		// Handle the message in a new goroutine to avoid blocking

		if actionData.Action == "80003" {
			gpsPacket, err := p.parseGPSPacket(message)
			if err != nil {
				logger.Error("error parsing GPS packet:", zap.Error(err))
				return err
			}
			asyncStore := dataStore.GetProcessChan()
			protoReply := gpsPacket.ToProtobufDeviceStatusGPS()
			asyncStore <- *protoReply
		} else if actionData.Action == "80004" {
			logger.Sugar().Infof("Alarm data received")
		} else {
			logger.Sugar().Infof("Unhandled action type: %s", actionData.Action)
		}
		// pass parameters to avoid race conditions
	}
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
