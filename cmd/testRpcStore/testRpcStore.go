package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
	"google.golang.org/grpc"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var logger = configuredLogger.Logger

type server struct {
	store.UnimplementedAvlDataStoreServer
}

var knownImei = map[string]types.DeviceType{
	"357454075177072": types.DeviceType_TELTONIKA,
	"356307043721579": types.DeviceType_WANWAY,
}

func (s *server) VerifyDevice(ctx context.Context, req *store.VerifyDeviceRequest) (*store.VerifyDeviceReply, error) {
	reply := store.VerifyDeviceReply{}
	reply.Imei = req.Imei
	reply.DeviceType = knownImei[req.Imei]
	return &reply, nil
}

func (s *server) SaveDeviceStatus(ctx context.Context, req *types.DeviceStatus) (*emptypb.Empty, error) {
	deviceType := knownImei[req.Imei]
	logger.Sugar().Infoln(deviceType, req.String())
	return &emptypb.Empty{}, nil
}

func main() {
	port := flag.Int("port", 0, "port for this server")
	flag.Parse()

	if port == nil || *port == 0 {
		fmt.Fprintln(os.Stderr, "Usage:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		logger.Sugar().Fatalf("failed to listen: %v", err)
	}

	logger.Sugar().Infoln("Listening on port ", *port)
	s := grpc.NewServer()
	server := &server{}
	store.RegisterAvlDataStoreServer(s, server)
	if err := s.Serve(lis); err != nil {
		logger.Sugar().Fatalf("failed to serve: %v", err)
	}
}
