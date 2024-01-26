package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/404minds/avl-receiver/internal/handlers"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
)

var logger = configuredLogger.Logger

func main() {
	var port = flag.Int("port", 9000, "Port to listen on")
	flag.Parse()

	if port == nil {
		log.Panic("Port not specified")
	}

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

		go handlers.TcpHandler.HandleConnection(conn)
	}
}
