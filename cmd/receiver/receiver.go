package main

import (
	"flag"
	"fmt"
	"net"
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

func startGrpcServer(port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.Sugar().Fatalf("Failed to listen on port %d: %v", port, err)
	}
	grpcServer := grpc.NewServer()
	serverInstance := &server{}

	store.RegisterAvlReceiverServiceServer(grpcServer, serverInstance)
	
	logger.Sugar().Infof("gRPC server listening on port %d", port)
	if err := grpcServer.Serve(listener); err != nil {
		logger.Sugar().Fatalf("Failed to serve gRPC on port %d: %v", port, err)
	}
}

func main() {
	var port = flag.Int("port", 21000, "Port to listen on")
	var grpcPort = flag.Int("grpcPort", 22000, "Port for gRPC server")
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
	go startGrpcServer(*grpcPort)

	// Keep the main function running
	select {}
}

type server struct {
	store.AvlReceiverServiceServer
}
