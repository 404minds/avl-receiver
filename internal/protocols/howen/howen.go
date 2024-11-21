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
		err := p.ConsumeMessage(conn, dataStore)
		logger.Sugar().Info(err)
		if err != nil {
			return err
		}

		return nil
	}
}

func (p *HOWENWS) ConsumeMessage(conn *websocket.Conn, dataStore store.Store) error {
	for {
		// Read message from WebSocket
		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.Sugar().Info("consumer message", err)
			return errors.Wrap(err, "error reading WebSocket message: ")
		}

		logger.Sugar().Info(string(message))

		// Unmarshal the message to check the action type
		var actionData ActionData
		if err := json.Unmarshal(message, &actionData); err != nil {
			return errors.Wrap(err, "error un marshaling action data: ")
		}

		// Check if action type is 80003 (GPS data)
		if actionData.Action == "80003" {
			// Parse the GPS data

			gpsPacket, err := p.parseGPSPacket(message)
			if err != nil {
				return errors.Wrap(err, "error parsing GPS packet: ")
			}
			asyncStore := dataStore.GetProcessChan()
			protoReply := gpsPacket.ToProtobufDeviceStatusGPS()
			asyncStore <- *protoReply
			// Process the parsed GPS packet (e.g., save to dataStore)
		} else if actionData.Action == "80004" {
			alarmPacket, err := p.parseAlarmMessage(message)
			if err != nil {
				return errors.Wrap(err, "error parsing Alarm packet: ")
			}
			asyncStore := dataStore.GetProcessChan()
			protoReply := alarmPacket.ToProtobufDeviceStatusAlarm()
			asyncStore <- *protoReply
		} else {
			logger.Sugar().Infof("Unhandled action type: %s", actionData.Action)
		}
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
