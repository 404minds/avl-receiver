package handlers

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"runtime/debug"
	"slices"
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
	deviceProtocol, ack, err := t.attemptDeviceLogin(reader, conn)
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

// func (t *TcpHandler) attemptDeviceLogin(reader *bufio.Reader) (protocol devices.DeviceProtocol, ack []byte, err error) {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			logger.Sugar().Error("Step 0 - Panic occurred during protocol login: ", r)
// 			err = fmt.Errorf("panic occurred during protocol login: %v", r)
// 		}
// 	}()

// 	logger.Sugar().Infoln("Step 1 - Starting attemptDeviceLogin")

// 	for i, protocolType := range t.allowedProtocols {
// 		logger.Sugar().Infof("Step 2.%d - Trying protocol type: %s", i+1, protocolType)

// 		protocol = devices.MakeProtocolForType(protocolType)
// 		logger.Sugar().Infof("Step 3.%d - Created protocol instance: %T", i+1, protocol)

// 		if protocol == nil {
// 			logger.Sugar().Errorf("Step 4.%d - Unsupported ProtocolType: %s", i+1, protocolType)
// 			continue
// 		}

// 		logger.Sugar().Infof("Step 5.%d - Attempting login with protocol: %s", i+1, protocolType)
// 		ack, bytesToSkip, err := protocol.Login(reader)
// 		logger.Sugar().Infof("Step 6.%d - Login response => Ack: %v, BytesToSkip: %d, Error: %v", i+1, ack, bytesToSkip, err)

// 		if err != nil {
// 			if errors.Is(err, errs.ErrUnknownProtocol) {
// 				logger.Sugar().Errorf("Step 7.%d - Unknown protocol error: %v", i+1, err)
// 				continue // try another protocol
// 			}
// 			logger.Sugar().Errorf("Step 8.%d - Error during login: %v", i+1, err)
// 			return nil, nil, err
// 		}

// 		deviceID := protocol.GetDeviceID()
// 		logger.Sugar().Infof("Step 9.%d - Retrieved Device ID: %s", i+1, deviceID)

// 		if deviceID == "" {
// 			logger.Sugar().Errorf("Step 10.%d - Device ID is empty after successful login", i+1)
// 			continue
// 		}

// 		logger.Info("Step 11 - Device identified",
// 			zap.String("protocol", protocolType.String()),
// 			zap.String("deviceID", deviceID),
// 			zap.Int("bytesToSkip", bytesToSkip),
// 		)

// 		if _, err := reader.Discard(bytesToSkip); err != nil {
// 			logger.Sugar().Errorf("Step 12.%d - Failed to discard %d bytes: %v", i+1, bytesToSkip, err)
// 			return nil, nil, err
// 		}

// 		deviceType, err := t.VerifyDevice(deviceID, protocol.GetProtocolType())
// 		logger.Sugar().Infof("Step 13.%d - Device type: %v, Error: %v", i+1, deviceType, err)

// 		if err != nil {
// 			if errors.Is(err, errs.ErrUnauthorizedDevice) {
// 				logger.Error("Step 14 - Device is not authorized",
// 					zap.String("deviceID", deviceID),
// 					zap.String("protocolType", protocol.GetProtocolType().String()),
// 				)
// 				return nil, nil, err
// 			}
// 			logger.Sugar().Errorf("Step 15.%d - Error verifying device: %v", i+1, err)
// 			return nil, nil, err
// 		}

// 		protocol.SetDeviceType(deviceType)
// 		logger.Info("Step 16 - Login successful",
// 			zap.String("deviceID", deviceID),
// 			zap.String("deviceType", deviceType.String()),
// 		)

// 		return protocol, ack, nil
// 	}

// 	logger.Sugar().Error("Step 17 - All protocols failed, unknown device type")
// 	return nil, nil, errs.ErrUnknownDeviceType
// }

func (t *TcpHandler) attemptDeviceLogin(reader *bufio.Reader, conn net.Conn) (protocol devices.DeviceProtocol, ack []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Sugar().Error("Step 0 - Panic occurred during protocol login: ", r)
			err = fmt.Errorf("panic occurred during protocol login: %v", r)
		}
	}()

	buffered := reader.Buffered()
	initialData := make([]byte, buffered)
	_, err = reader.Peek(buffered)
	if err != nil {
		return nil, nil, fmt.Errorf("error peeking initial data: %v", err)
	}
	copy(initialData, initialData) // Ensure we have a copy

	header, headerErr := reader.Peek(8)
	logger.Sugar().Infoln("Step 0 OUTSIDE FOR LOOP", t, "reader", header, headerErr)

	logger.Sugar().Infoln("Step 1 - Starting attemptDeviceLogin")

	for i, protocolType := range t.allowedProtocols {

		copiedReader := bufio.NewReader(
			io.MultiReader(
				bytes.NewReader(initialData),
				conn,
			),
		)
		// Create a new reader for each protocol attempt using the copied data
		header, headerErr := reader.Peek(2)
		logger.Sugar().Infoln("Step 1.13000 - INSIDE FOR LOOP", t, "reader", header, headerErr)

		logger.Sugar().Infof("Step 2.%d - Trying protocol type: %s", i+1, protocolType)

		protocol = devices.MakeProtocolForType(protocolType)
		logger.Sugar().Infof("Step 3.%d - Created protocol instance: %T", i+1, protocol)

		if protocol == nil {
			logger.Sugar().Errorf("Step 4.%d - Unsupported ProtocolType: %s", i+1, protocolType)
			continue
		}

		logger.Sugar().Infof("Step 5.%d - Attempting login with protocol: %s", i+1, protocolType)
		ack, bytesToSkip, err := protocol.Login(reader)
		logger.Sugar().Infof("Step 6.%d - Login response => Ack: %v, BytesToSkip: %d, Error: %v", i+1, ack, bytesToSkip, err)

		if err != nil {
			if errors.Is(err, errs.ErrUnknownProtocol) || errors.Is(err, errs.ErrTR06InvalidLoginInfo) || errors.Is(err, errs.ErrGT06InvalidLoginInfo) || errors.Is(err, io.EOF) {
				logger.Sugar().Warnf("Step 7.%d - Protocol detection error: %v, trying next protocol", i+1, err)
				continue // try another protocol
			}
			logger.Sugar().Errorf("Step 8.%d - Fatal error during login: %v", i+1, err)
			return nil, nil, err
		}

		deviceID := protocol.GetDeviceID()
		logger.Sugar().Infof("Step 9.%d - Retrieved Device ID: %s", i+1, deviceID)

		if deviceID == "" {
			logger.Sugar().Errorf("Step 10.%d - Device ID is empty after successful login", i+1)
			continue
		}

		logger.Info("Step 11 - Device identified",
			zap.String("protocol", protocolType.String()),
			zap.String("deviceID", deviceID),
			zap.Int("bytesToSkip", bytesToSkip),
		)

		if bytesToSkip > 0 {
			if _, err := reader.Discard(bytesToSkip); err != nil {
				logger.Sugar().Errorf("Step 12.%d - Failed to discard %d bytes: %v", i+1, bytesToSkip, err)
				return nil, nil, err
			}
		}

		deviceType, err := t.VerifyDevice(deviceID, protocol.GetProtocolType())
		logger.Sugar().Infof("Step 13.%d - Device type: %v, Error: %v", i+1, deviceType, err)

		if err != nil {
			if errors.Is(err, errs.ErrUnauthorizedDevice) {
				logger.Error("Step 14 - Device is not authorized",
					zap.String("deviceID", deviceID),
					zap.String("protocolType", protocol.GetProtocolType().String()),
				)
				return nil, nil, err
			}
			logger.Sugar().Errorf("Step 15.%d - Error verifying device: %v", i+1, err)
			return nil, nil, err
		}

		protocol.SetDeviceType(deviceType)
		logger.Info("Step 16 - Login successful",
			zap.String("deviceID", deviceID),
			zap.String("deviceType", deviceType.String()),
		)

		return protocol, ack, nil
	}

	logger.Sugar().Error("Step 17 - All protocols failed, unknown device type")
	return nil, nil, errs.ErrUnknownDeviceType
}

// Getter for connection info by IMEI

func (t *TcpHandler) GetConnInfoByIMEI(imei string) (DeviceConnectionInfo, bool) {
	info, exists := t.imeiToConnMap[imei]
	return info, exists
}
