package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/404minds/avl-receiver/internal/handlers"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var logger = configuredLogger.Logger

func main() {
	var port = flag.Int("port", 9000, "Port to listen on")
	var remoteStoreAddr = flag.String("remoteStoreAddr", "", "Address of the remote store")
	flag.Parse()

	if *port == 0 || *remoteStoreAddr == "" {
		fmt.Fprintln(os.Stderr, "Usage:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	storeConn, err := grpc.Dial(*remoteStoreAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Sugar().Fatalf("did not connect: %v", err)
	}
	defer storeConn.Close()

	remoteStoreClient := store.NewAvlDataStoreClient(storeConn)
	tcpHandler := handlers.NewTcpHandler(remoteStoreClient)

	listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", *port))
	if err != nil {
		logger.Sugar().Errorf("Error listening on port %d", *port)
		logger.Error(err.Error())
		return
	}

	logger.Sugar().Infof("Listening on port %d", *port)
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting a new connection:", err.Error())
			continue
		}
		logger.Sugar().Infof("New connection from %s", conn.RemoteAddr().String())

		go tcpHandler.HandleConnection(conn)
	}
}
