package handlers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	errs "github.com/404minds/avl-receiver/internal/errors"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	devices "github.com/404minds/avl-receiver/internal/protocols"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
	"go.uber.org/zap"
)

var logger = configuredLogger.Logger

type DeviceConnectionInfo struct {
	Conn     net.Conn
	Protocol devices.DeviceProtocol
}

type TcpHandler struct {
	mu                sync.RWMutex
	connToProtocolMap map[string]devices.DeviceProtocol // make this an LRU cache to evict stale connections
	allowedProtocols  []types.DeviceProtocolType
	connToStoreMap    map[string]store.Store
	remoteStoreClient store.CustomAvlDataStoreClient
	storeType         string
	imeiToConnMap     map[string]DeviceConnectionInfo
}

func (t *TcpHandler) HandleConnection(conn net.Conn) {
	var remoteAddr = conn.RemoteAddr().String()

	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			return
		}
	}(conn)

	err := conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return
	}
	err = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return
	}
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('*')
	if err != nil {
		// return nil, 0, (err, "failed to read login packet")
	}
	line = strings.TrimSpace(line) // remove newline and any trailing whitespace

	logger.Sugar().Infoln("bahar line", line)
	deviceProtocol, ack, err := t.attemptDeviceLogin(reader)
	if err != nil {
		logger.Error("failed to identify device", zap.String("remoteAddr", remoteAddr), zap.Error(err))
		return
	}

	// Lock for map writes
	t.mu.Lock()
	t.connToProtocolMap[remoteAddr] = deviceProtocol
	deviceID := deviceProtocol.GetDeviceID()

	if deviceID != "" {
		t.imeiToConnMap[deviceID] = DeviceConnectionInfo{
			Conn:     conn,
			Protocol: deviceProtocol,
		}
		logger.Sugar().Infof("Mapped deviceID %s to connection %v", deviceID, remoteAddr)
	}
	t.mu.Unlock()

	dataStore := t.makeAsyncStore(deviceProtocol)

	// Start processing goroutines
	// Start processing goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Sugar().Errorf("Recovered from panic in Process goroutine: %v, \n Stack trace %s", r, debug.Stack())
			}
		}()
		dataStore.Process(ctx)
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Sugar().Errorf("Recovered from panic in Response goroutine: %v, \n Stack trace %s", r, debug.Stack())
			}
		}()
		dataStore.Response(ctx)
	}()

	defer func() {
		dataStore.GetCloseChan() <- true
		dataStore.GetCloseResponseChan() <- true
		close(dataStore.GetCloseChan())
		close(dataStore.GetCloseResponseChan())

		t.mu.Lock()
		defer t.mu.Unlock()

		// Clean up all maps
		delete(t.connToProtocolMap, remoteAddr)
		delete(t.connToStoreMap, remoteAddr)

		// More efficient cleanup using reverse mapping
		for imei, info := range t.imeiToConnMap {
			if info.Conn == conn {
				delete(t.imeiToConnMap, imei)
				break
			}
		}
	}()

	// Lock for store map update
	t.mu.Lock()
	t.connToStoreMap[remoteAddr] = dataStore
	t.mu.Unlock()

	if _, err = conn.Write(ack); err != nil {
		logger.Error("Error writing login ack", zap.Error(err))
		return
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
					logger.Error("failed to refresh read deadline", zap.Error(err))
					return
				}
				if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
					logger.Error("failed to refresh write deadline", zap.Error(err))
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

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
		ProcessChan:       make(chan *types.DeviceStatus, 200),
		ResponseChan:      make(chan *types.DeviceResponse, 200),
		CloseChan:         make(chan bool, 1),
		CloseResponseChan: make(chan bool, 1),
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
		File:              file,
		ProcessChan:       make(chan *types.DeviceStatus, 200),
		ResponseChan:      make(chan *types.DeviceResponse, 200),
		CloseChan:         make(chan bool, 200),
		CloseResponseChan: make(chan bool, 200),
		DeviceID:          deviceIdentifier,
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
				// continue // try another device
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
