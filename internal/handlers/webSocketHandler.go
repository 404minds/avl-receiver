package handlers

import (
	"context"
	"errors"
	devices "github.com/404minds/avl-receiver/internal/protocols"
	"github.com/404minds/avl-receiver/internal/protocols/howen"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
	"github.com/gorilla/websocket"
	"os"
)

type WebSocketHandler struct {
	connToProtocolMap map[string]devices.DeviceProtocol
	allowedProtocols  []types.DeviceProtocolType
	remoteStoreClient store.CustomAvlDataStoreClient
	storeType         string
	connToStoreMap    map[string]store.Store
}

// HandleMessage processes the incoming message and parses it based on action type
func (w *WebSocketHandler) HandleMessage(conn *websocket.Conn) {
	logger.Sugar().Info("creating data store")

	deviceProtocol := &howen.HOWENWS{DeviceType: types.DeviceType_HOWEN}
	dataStore := w.makeAsyncStore(deviceProtocol)

	ctx, _ := context.WithCancel(context.Background())
	logger.Sugar().Info("starting data store process")
	go dataStore.Process(ctx)

	defer func() {
		dataStore.GetCloseChan() <- true
		if conn != nil {
			conn.Close()
		}
	}()

	err := deviceProtocol.ConsumeConnection(conn, dataStore)
	if err != nil {
		if websocket.IsUnexpectedCloseError(err) {
			logger.Sugar().Error("WebSocket connection closed unexpectedly:", err)
		} else {
			logger.Sugar().Error("WebSocket connection handling error:", err)
		}
	}
}

func (w *WebSocketHandler) makeAsyncStore(deviceProtocol devices.DeviceProtocol) store.Store {
	if w.storeType == "local" {
		if err := os.Mkdir("./logs", os.ModeDir); err == nil || errors.Is(err, os.ErrExist) {
			return makeJsonStore("./logs", deviceProtocol.GetDeviceID())
		}
	} else if w.storeType == "remote" {
		return makeRemoteRpcStore(w.remoteStoreClient)
	} else {
		panic("Invalid store type")
	}
	return nil
}
