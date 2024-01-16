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
	var port = flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	if port == nil {
		log.Panic("Port not specified")
	}

	ln, err := net.Listen("tcp4", fmt.Sprintf(":%d", *port))
	if err != nil {
		logger.Sugar().Errorf("Error listening on port %d", *port)
		logger.Error(err.Error())
		return
	}

	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting a new connection:", err.Error())
			continue
		}

		go handlers.TcpHandler.HandleConnection(conn)
	}
}
