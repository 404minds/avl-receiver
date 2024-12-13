package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/404minds/avl-receiver/internal/handlers"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

var logger = configuredLogger.Logger

type server struct {
	store.UnimplementedAvlReceiverServiceServer
	tcpHandler *handlers.TcpHandler
}

func startGrpcServer(port int, tcpHandler *handlers.TcpHandler) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.Sugar().Fatalf("Failed to listen on port %d: %v", port, err)
	}
	s := grpc.NewServer()
	serverInstance := &server{
		tcpHandler: tcpHandler,
	}

	store.RegisterAvlReceiverServiceServer(s, serverInstance)

	logger.Sugar().Infof("gRPC server listening on port %d", port)
	if err := s.Serve(listener); err != nil {
		logger.Sugar().Fatalf("Failed to serve gRPC on port %d: %v", port, err)
	}
}

func main() {
	var port = flag.Int("port", 21000, "Port to listen on")
	var grpcPort = flag.Int("grpcPort", 15000, "Port for gRPC server")
	var remoteStoreAddr = flag.String("remoteStoreAddr", "", "Address of the remote store")
	var storeType = flag.String("storeType", "remote", "Store type - one of local or remote")
	flag.Parse()

	if *port == 0 || *remoteStoreAddr == "" || *storeType == "" {
		_, _ = fmt.Fprintln(os.Stderr, "Usage:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	storeConn, err := grpc.Dial(*remoteStoreAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Sugar().Fatalf("did not connect: %v", err)
	}
	defer storeConn.Close()

	go func() {
		time.Sleep(5 * time.Second)
		if storeConn.GetState() != connectivity.Ready {
			logger.Sugar().Errorf("Connection to gRPC server %s not ready", *remoteStoreAddr)
		} else {
			logger.Sugar().Infof("Connected to gRPC server %s", *remoteStoreAddr)
		}
	}()

	remoteStoreClient := store.NewCustomAvlDataStoreClient(storeConn)
	tcpHandler := handlers.NewTcpHandler(*remoteStoreClient, *storeType)
	websocketHandler := handlers.NewWebSocketHandler(*remoteStoreClient, *storeType)

	// Start TCP Server
	go func() {
		listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", *port))
		if err != nil {
			logger.Sugar().Errorf("Error listening on port %d", *port)
			logger.Error(err.Error())
			return
		}
		logger.Sugar().Infof("TCP server listening on port %d", *port)
		defer listener.Close()

		for {
			conn, err := listener.Accept()
			if err != nil {
				logger.Sugar().Errorf("Error accepting a new connection: %v", err)
				continue
			}
			logger.Sugar().Infof("New connection from %s", conn.RemoteAddr().String())
			go tcpHandler.HandleConnection(conn)
		}
	}()

	// Start gRPC Server
	go startGrpcServer(*grpcPort, &tcpHandler)

	// Start WebSocket connection for real-time data
	go startWebSocket(&websocketHandler)

	// Keep the main function running
	select {}
}

func (s *server) SendCommand(ctx context.Context, req *store.SendCommandRequestAVL) (*store.SendCommandResponseAVL, error) {
	// Access the imeiToConnMap through the TcpHandler
	info, exists := s.tcpHandler.GetConnInfoByIMEI(req.Imei)
	if !exists {
		return &store.SendCommandResponseAVL{
			Success: false,
			Message: "Device not found",
		}, nil
	}

	conn := info.Conn
	protocol := info.Protocol

	logger.Sugar().Infof("Sending command to remote device %s", conn.RemoteAddr().String())
	logger.Sugar().Info("protocol: ", protocol)
	// Prepare to send the command to the device
	writer := bufio.NewWriter(conn) // You can adjust this as needed
	err := protocol.SendCommandToDevice(writer, req.Command)
	if err != nil {
		return &store.SendCommandResponseAVL{
			Success: false,
			Message: "Failed to send command: " + err.Error(),
		}, nil
	}

	// Flush the writer to ensure the command is sent
	if err := writer.Flush(); err != nil {
		return &store.SendCommandResponseAVL{
			Success: false,
			Message: "Failed to flush writer: " + err.Error(),
		}, nil
	}

	return &store.SendCommandResponseAVL{
		Success: true,
		Message: "Command sent successfully",
	}, nil
}

const (
	httpLoginURL = "https://vss.howentech.com/vss/user/apiLogin.action"
	wsURL        = "ws://47.252.16.64:36300"
)

type LoginResponse struct {
	Status int         `json:"status"`
	Msg    string      `json:"msg"`
	Error  interface{} `json:"error"`
	Data   struct {
		Token string `json:"token"`
		PID   string `json:"pid"`
	} `json:"data"`
	Count int `json:"count"`
}

type LoginMessage struct {
	Action  string `json:"action"`
	Payload struct {
		Username string `json:"username"`
		Token    string `json:"token"`
		PID      string `json:"pid"`
	} `json:"payload"`
}

type SubscribeMessage struct {
	Action string `json:"action"`
}

func getAuthToken(username, password string) (string, string, error) {
	data := "username=" + username + "&password=" + password
	req, err := http.NewRequest("POST", httpLoginURL, bytes.NewBufferString(data))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var loginResponse LoginResponse
	err = json.Unmarshal(body, &loginResponse)
	if err != nil {
		return "", "", err
	}

	return loginResponse.Data.Token, loginResponse.Data.PID, nil
}

func startWebSocket(wsHandler *handlers.WebSocketHandler) {
	token, pid, err := getAuthToken("INEITest", "964b4035ac5a3987036692d517eaf7fb")
	if err != nil {
		log.Fatal("Error during HTTP login:", err)
		return
	}

	for {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			logger.Sugar().Error("Error connecting to WebSocket:", err)
			time.Sleep(5 * time.Second) // Retry after delay
			continue
		}

		defer conn.Close()

		// Send login message
		login := LoginMessage{
			Action: "80000",
			Payload: struct {
				Username string `json:"username"`
				Token    string `json:"token"`
				PID      string `json:"pid"`
			}{
				Username: "INEITest",
				Token:    token,
				PID:      pid,
			},
		}

		if err := conn.WriteJSON(login); err != nil {
			logger.Sugar().Error("Error sending login message:", err)
			conn.Close()
			continue
		}

		logger.Sugar().Info("Login message sent")

		// Wait for response
		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.Sugar().Error("Error reading login response:", err)
			conn.Close()
			continue
		}

		logger.Sugar().Info("Received login response:", string(message))

		// Handle incoming WebSocket messages
		wsHandler.HandleMessage(conn)
	}
}
