package handlers

import (
	"bufio"
	"io"
	"net"
	"os"

	devices "github.com/404minds/avl-receiver/internal/devices"
	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
)

var logger = configuredLogger.Logger

const BUFFER_SIZE = 256 // bytes

type tcpHandler struct {
	connToProtocolMap     map[string]devices.DeviceProtocol // make this an LRU cache to evict stale connections
	registeredDeviceTypes []devices.AVLDeviceType
	connToStoreMap        map[string]store.Store
}

func (t *tcpHandler) HandleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	deviceProtocol, ack, err := t.attemptDeviceLogin(reader)
	if err != nil {
		logger.Error("Failed to connect device")
		logger.Error(err.Error())
		return
	}

	t.connToProtocolMap[conn.RemoteAddr().String()] = deviceProtocol
	dataStore := makeJsonStore(deviceProtocol.GetDeviceIdentifier())
	go dataStore.Process()
	defer func() { dataStore.GetCloseChan() <- true }()

	t.connToStoreMap[conn.RemoteAddr().String()] = dataStore
	conn.Write(ack)

	writer := bufio.NewWriter(conn)
	err = deviceProtocol.ConsumeStream(reader, writer, dataStore.GetProcessChan())
	if err != nil && err != io.EOF {
		logger.Sugar().Errorf("Error reading from connection %s", conn.RemoteAddr().String())
		logger.Error(err.Error())
		return
	} else if err == io.EOF {
		logger.Sugar().Infof("Connection %s closed", conn.RemoteAddr().String())
		return
	}
}

func makeJsonStore(deviceIdentifier string) store.Store {
	file, err := os.CreateTemp("", deviceIdentifier+".json")
	if err != nil {
		logger.Error("failed to open file to store data")
		logger.Panic(err.Error())
	}
	logger.Sugar().Infof("[deviceId: %s] Created json file store at %s", deviceIdentifier, file.Name())

	return &store.JsonLinesStore{
		File:        file,
		ProcessChan: make(chan interface{}, 200),
		CloseChan:   make(chan bool, 200),
	}
}

func (t *tcpHandler) attemptDeviceLogin(reader *bufio.Reader) (devices.DeviceProtocol, []byte, error) {
	for _, deviceType := range t.registeredDeviceTypes {
		protocol := deviceType.GetProtocol()
		ack, bytesConsumed, err := protocol.Login(reader)

		if err != nil {
			continue // try another device
		} else {
			// discard bytes consumed by login to since we already have a final protocol that worked
			logger.Sugar().Infof("Device identified to be of type %s with identifier %s", deviceType.String(), protocol.GetDeviceIdentifier())
			if _, err := reader.Discard(bytesConsumed); err != nil {
				return nil, nil, err
			}
			return protocol, ack, nil
		}
	}

	return nil, nil, errs.ErrUnknownDeviceType
}
