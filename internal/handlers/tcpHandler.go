package handlers

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path"
	"slices"

	devices "github.com/404minds/avl-receiver/internal/devices"
	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
)

var logger = configuredLogger.Logger

type TcpHandler struct {
	connToProtocolMap   map[string]devices.DeviceProtocol // make this an LRU cache to evict stale connections
	registeredProtocols []types.DeviceProtocolType
	connToStoreMap      map[string]store.Store
	remoteStoreClient   store.AvlDataStoreClient
}

func (t *TcpHandler) HandleConnection(conn net.Conn) {
	defer conn.Close()
	defer func() {
		delete(t.connToProtocolMap, conn.RemoteAddr().String())
		delete(t.connToStoreMap, conn.RemoteAddr().String())
	}()

	reader := bufio.NewReader(conn)

	deviceProtocol, ack, err := t.attemptDeviceLogin(reader)
	if err != nil {
		logger.Sugar().Errorf("failed to identify device from %s : %s", conn.RemoteAddr().String(), err)
		return
	}

	t.connToProtocolMap[conn.RemoteAddr().String()] = deviceProtocol
	// dataStore := makeJsonStore(t.dataDir, deviceProtocol.GetDeviceIdentifier())
	dataStore := makeRemoteRpcStore(t.remoteStoreClient)
	go dataStore.Process()
	defer func() { dataStore.GetCloseChan() <- true }()

	t.connToStoreMap[conn.RemoteAddr().String()] = dataStore
	_, err = conn.Write(ack)
	if err != nil {
		logger.Sugar().Error("Error while writing login ack", err)
		return
	}

	err = deviceProtocol.ConsumeStream(reader, conn, dataStore.GetProcessChan())
	if err != nil && err != io.EOF {
		logger.Sugar().Errorf("Error reading from connection %s", conn.RemoteAddr().String(), err)
		//logger.Error(err.Error())
		return
	} else if err == io.EOF {
		logger.Sugar().Infof("Connection %s closed", conn.RemoteAddr().String())
		return
	}
}

func makeRemoteRpcStore(remoteStoreClient store.AvlDataStoreClient) store.Store {
	return &store.RemoteRpcStore{
		ProcessChan:       make(chan types.DeviceStatus, 200),
		CloseChan:         make(chan bool, 200),
		RemoteStoreClient: remoteStoreClient,
	}
}

func makeJsonStore(datadir string, deviceIdentifier string) store.Store {
	file, err := os.OpenFile(path.Join(datadir, deviceIdentifier+".json"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("failed to open file to store data")
		logger.Panic(err.Error())
	}
	logger.Sugar().Infof("[deviceId: %s] Created json file store at %s", deviceIdentifier, file.Name())

	return &store.JsonLinesStore{
		File:        file,
		ProcessChan: make(chan types.DeviceStatus, 200),
		CloseChan:   make(chan bool, 200),
	}
}

func (t *TcpHandler) attemptDeviceLogin(reader *bufio.Reader) (devices.DeviceProtocol, []byte, error) {
	for _, protocolType := range t.registeredProtocols {
		protocol := devices.MakeProtocolForType(protocolType)
		ack, bytesToSkip, err := protocol.Login(reader)

		if err != nil {
			if errors.Is(err, errs.ErrUnknownDeviceType) {
				continue // try another device
			} else {
				return nil, nil, err
			}
		} else {
			logger.Sugar().Infof("Protocol identified to be of type %s with identifier %s, bytes to skip %d", protocolType.String(), protocol.GetDeviceIdentifier(), bytesToSkip)
			if _, err := reader.Discard(bytesToSkip); err != nil {
				return nil, nil, err
			}

			reply, err := t.remoteStoreClient.VerifyDevice(context.Background(), &store.VerifyDeviceRequest{
				Imei: protocol.GetDeviceIdentifier(),
			})
			if err != nil {
				logger.Sugar().Errorf("failed to verify device %s", protocol.GetDeviceIdentifier())
			}
			if reply.GetImei() != protocol.GetDeviceIdentifier() || slices.Contains(devices.GetDeviceTypesForProtocol(protocolType), reply.GetDeviceType()) == false {
				logger.Sugar().Infof("Device %s is not authorized to connect", protocol.GetDeviceIdentifier())
				return nil, nil, errs.ErrUnauthorizedDevice
			} else {
				protocol.SetDeviceType(reply.GetDeviceType())
				logger.Sugar().Infof("Login successful for device %s of type %s", protocol.GetDeviceIdentifier(), protocol.GetDeviceType())
			}

			return protocol, ack, nil
		}
	}

	return nil, nil, errs.ErrUnknownDeviceType
}
