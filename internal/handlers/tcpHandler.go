package handlers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net"
	"os"
	"path"
	"slices"

	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	devices "github.com/404minds/avl-receiver/internal/protocols"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
)

var logger = configuredLogger.Logger

type DeviceConnectionInfo struct {
	Conn     net.Conn
	Protocol devices.DeviceProtocol
}

type TcpHandler struct {
	connToProtocolMap map[string]devices.DeviceProtocol // make this an LRU cache to evict stale connections
	allowedProtocols  []types.DeviceProtocolType
	connToStoreMap    map[string]store.Store
	remoteStoreClient store.CustomAvlDataStoreClient
	storeType         string
	imeiToConnMap     map[string]DeviceConnectionInfo
}

func (t *TcpHandler) HandleConnection(conn net.Conn) {
	var remoteAddr = conn.RemoteAddr().String()

	defer conn.Close()
	defer func() {
		delete(t.connToProtocolMap, remoteAddr)
		delete(t.connToStoreMap, remoteAddr)
		for imei, info := range t.imeiToConnMap {
			if info.Conn == conn {
				delete(t.imeiToConnMap, imei)
				break
			}
		}
	}()

	reader := bufio.NewReader(conn)
	deviceProtocol, ack, err := t.attemptDeviceLogin(reader)
	if err != nil {
		logger.Error("failed to identify device", zap.String("remoteAddr", remoteAddr), zap.Error(err))
		return
	}

	t.connToProtocolMap[remoteAddr] = deviceProtocol
	deviceID := deviceProtocol.GetDeviceID()
	// Store the IMEI, connection, and protocol type in the combined map
	if deviceID != "" {
		t.imeiToConnMap[deviceID] = DeviceConnectionInfo{
			Conn:     conn,
			Protocol: deviceProtocol,
		}
		logger.Sugar().Infof("Mapped deviceID %s to connection %v and protocol %v", deviceID, conn.RemoteAddr().String(), deviceProtocol)
	}

	dataStore := t.makeAsyncStore(deviceProtocol)
	go dataStore.Process()
	go dataStore.Response()
	defer func() { dataStore.GetCloseChan() <- true }()

	t.connToStoreMap[remoteAddr] = dataStore
	_, err = conn.Write(ack)
	if err != nil {
		logger.Error("Error while writing login ack", zap.Error(err))
		return
	}

	err = deviceProtocol.ConsumeStream(reader, conn, dataStore)
	if err != nil && err != io.EOF {
		logger.Error("Failure while reading from stream", zap.String("remoteAddr", remoteAddr), zap.Error(err))
		return
	} else if err == io.EOF {
		logger.Sugar().Infof("Connection %s closed", conn.RemoteAddr().String())
		return
	}
}

func (t *TcpHandler) makeAsyncStore(deviceProtocol devices.DeviceProtocol) store.Store {
	if t.storeType == "local" {
		if err := os.Mkdir("./logs", os.ModeDir); err == nil || errors.Is(err, os.ErrExist) {
			return makeJsonStore("./logs", deviceProtocol.GetDeviceID())
		}
	} else if t.storeType == "remote" {
		return makeRemoteRpcStore(t.remoteStoreClient)
	} else {
		panic("Invalid store type")
	}
	return nil
}

func makeRemoteRpcStore(remoteStoreClient store.CustomAvlDataStoreClient) store.Store {
	return &store.RemoteRpcStore{
		ProcessChan:       make(chan types.DeviceStatus, 200),
		ResponseChan:      make(chan types.DeviceResponse, 200),
		CloseChan:         make(chan bool, 200),
		RemoteStoreClient: remoteStoreClient,
	}
}

func (t *TcpHandler) VerifyDevice(deviceID string, detectedProtocol types.DeviceProtocolType) (types.DeviceType, error) {
	if t.storeType == "local" {
		return devices.GetDeviceTypesForProtocol(detectedProtocol)[0], nil
	} else {
		req := store.VerifyDeviceRequest{Imei: deviceID}
		reply, err := t.remoteStoreClient.VerifyDevice(context.Background(), &req)
		if err != nil {
			logger.Error("Failed to verify device", zap.String("deviceID", deviceID), zap.String("detectedProtocol", detectedProtocol.String()), zap.Error(err))
			return 0, err
		}

		if reply.GetImei() != deviceID ||
			!slices.Contains(devices.GetDeviceTypesForProtocol(detectedProtocol), reply.GetDeviceType()) {
			return 0, errs.ErrUnauthorizedDevice
		}
		return reply.GetDeviceType(), nil
	}
}

func makeJsonStore(destDir string, deviceIdentifier string) store.Store {
	file, err := os.OpenFile(path.Join(destDir, deviceIdentifier+".json"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error("failed to open file to store data")
		logger.Panic(err.Error())
	}
	logger.Sugar().Infof("[deviceId: %s] Created json file store at %s", deviceIdentifier, file.Name())

	return &store.JsonLinesStore{
		File:         file,
		ProcessChan:  make(chan types.DeviceStatus, 200),
		ResponseChan: make(chan types.DeviceResponse, 200),
		CloseChan:    make(chan bool, 200),
		DeviceID:     deviceIdentifier,
	}
}

func (t *TcpHandler) attemptDeviceLogin(reader *bufio.Reader) (protocol devices.DeviceProtocol, ack []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Sugar().Error("Panic occurred during protocol login: ", r)
			err = fmt.Errorf("panic occurred during protocol login: %v", r)
		}
	}()

	for _, protocolType := range t.allowedProtocols {
		logger.Sugar().Info("Attempting device login with ProtocolType: ", protocolType)
		protocol = devices.MakeProtocolForType(protocolType)
		logger.Sugar().Info("Created Protocol: ", protocol)

		if protocol == nil {
			logger.Sugar().Error("Unsupported ProtocolType: ", protocolType)
			continue
		}

		logger.Sugar().Info("Attempting to login with protocol: ", protocolType)
		ack, bytesToSkip, err := protocol.Login(reader)
		logger.Sugar().Infof("Acknowledgement: %v for bytes to skip: %d and error: %v", ack, bytesToSkip, err)

		if err != nil {
			if errors.Is(err, errs.ErrUnknownProtocol) {
				logger.Sugar().Error("Unknown protocol error: ", err)
				continue // try another device
			}
			logger.Sugar().Error("Error during login: ", err)
			return nil, nil, err
		}

		// Only call GetDeviceID after a successful login
		deviceID := protocol.GetDeviceID()
		logger.Sugar().Infof("Device ID: %s", deviceID)

		if deviceID == "" {
			logger.Error("Device ID is empty after successful login")
			continue
		}

		logger.Info("Device identified", zap.String("protocol", protocolType.String()), zap.String("deviceID", deviceID), zap.Int("bytesToSkip", bytesToSkip))
		if _, err := reader.Discard(bytesToSkip); err != nil {
			logger.Sugar().Error("Error discarding bytes: ", err)
			return nil, nil, err
		}

		deviceType, err := t.VerifyDevice(deviceID, protocol.GetProtocolType())
		logger.Sugar().Info("device Type: ", deviceType, " error: ", err)
		if err != nil {
			if errors.Is(err, errs.ErrUnauthorizedDevice) {
				logger.Error("Device is not authorized", zap.String("deviceID", deviceID), zap.String("protocolType", protocol.GetProtocolType().String()))
				return nil, nil, err
			}
			logger.Sugar().Error("Error verifying device: ", err)
			return nil, nil, err
		}

		protocol.SetDeviceType(deviceType)
		logger.Info("Login successful", zap.String("deviceID", deviceID), zap.String("deviceType", deviceType.String()))
		return protocol, ack, nil
	}

	logger.Sugar().Error("All protocols failed, unknown device type")
	return nil, nil, errs.ErrUnknownDeviceType
}

// Getter for connection info by IMEI
func (t *TcpHandler) GetConnInfoByIMEI(imei string) (DeviceConnectionInfo, bool) {
	info, exists := t.imeiToConnMap[imei]
	return info, exists
}
