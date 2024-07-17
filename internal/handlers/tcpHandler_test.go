package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"testing"

	errs "github.com/404minds/avl-receiver/internal/errors"
	"github.com/404minds/avl-receiver/internal/protocols/fm1200"
	"github.com/404minds/avl-receiver/internal/protocols/gt06"
	"github.com/404minds/avl-receiver/internal/store"
	"github.com/404minds/avl-receiver/internal/types"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type mockRemoteDataStore struct {
	Imei       string
	DeviceType types.DeviceType
}

func (s *mockRemoteDataStore) VerifyDevice(ctx context.Context, in *store.VerifyDeviceRequest, opts ...grpc.CallOption) (*store.VerifyDeviceReply, error) {
	return &store.VerifyDeviceReply{
		Imei:       s.Imei,
		DeviceType: s.DeviceType,
	}, nil
}

func (s *mockRemoteDataStore) SaveDeviceStatus(ctx context.Context, in *types.DeviceStatus, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func TestTeltonikaDeviceLogin(t *testing.T) {
	buf, _ := hex.DecodeString("000F333536333037303433373231353739")

	reader := bufio.NewReader(bytes.NewReader(buf))
	handler := NewTcpHandler(&mockRemoteDataStore{
		Imei:       "356307043721579",
		DeviceType: types.DeviceType_TELTONIKA,
	}, "")
	protocol, ack, err := handler.attemptDeviceLogin(reader)

	assert.NoError(t, err, "device login should succeed")
	assert.IsType(t, &fm1200.FM1200Protocol{}, protocol, "protocol should be of type FM1200Protocol")
	assert.Equal(t, "356307043721579", protocol.GetDeviceID(), "imei should be parsed correctly")
	assert.Equal(t, []byte{0x01}, ack, "ack should be 0x01")
}

func TestWanwayDeviceLogin(t *testing.T) {
	hexString := "78781101075253367890024270003201000512790D0A"
	buf, _ := hex.DecodeString(hexString)

	reader := bufio.NewReader(bytes.NewReader(buf))
	handler := NewTcpHandler(&mockRemoteDataStore{
		Imei:       "752533678900242",
		DeviceType: types.DeviceType_WANWAY,
	}, "")
	protocol, ack, err := handler.attemptDeviceLogin(reader)

	assert.NoError(t, err, "device login should succeed")
	assert.IsType(t, &gt06.GT06Protocol{}, protocol, "protocol should be of type GT06Protocol")
	assert.Equal(t, "752533678900242", protocol.GetDeviceID(), "imei should be parsed correctly")
	assert.Equal(t, []byte{0x78, 0x78, 0x11, 0x01, 0x00, 0x05, 0x12, 0x79, 0x0d, 0x0a}, ack, "login ack should be of the format as GT06 expects")
}

func TestUnknownDeviceLogin(t *testing.T) {
	buf, _ := hex.DecodeString("7676fafafafa")
	reader := bufio.NewReader(bytes.NewReader(buf))
	handler := NewTcpHandler(&mockRemoteDataStore{}, "")
	protocol, ack, err := handler.attemptDeviceLogin(reader)

	assert.Nil(t, protocol, "protocol should be nil")
	assert.Nil(t, ack, "ack should be nil")
	assert.ErrorIs(t, err, errs.ErrUnknownDeviceType, "error should be ErrUnknownDevice")
}
