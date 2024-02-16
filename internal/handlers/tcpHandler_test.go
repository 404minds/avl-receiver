package handlers

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/404minds/avl-receiver/internal/devices/teltonika"
	"github.com/404minds/avl-receiver/internal/devices/wanway"
	errs "github.com/404minds/avl-receiver/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestTeltonikaDeviceLogin(t *testing.T) {
	buf, _ := hex.DecodeString("000F333536333037303433373231353739")

	reader := bufio.NewReader(bytes.NewReader(buf))
	handler := NewTcpHandler("")
	protocol, ack, err := handler.attemptDeviceLogin(reader)

	assert.NoError(t, err, "device login should succeed")
	assert.IsType(t, &teltonika.TeltonikaProtocol{}, protocol, "protocol should be of type TeltonikaProtocol")
	assert.Equal(t, "356307043721579", protocol.GetDeviceIdentifier(), "imei should be parsed correctly")
	assert.Equal(t, []byte{0x01}, ack, "ack should be 0x01")
}

func TestWanwayDeviceLogin(t *testing.T) {
	// buf, _ := hex.DecodeString("78781101035745407517707205184dd80001bb1a0d0a")
	buf, _ := hex.DecodeString("78781101035745407517707205184dd8000191400d0a")

	reader := bufio.NewReader(bytes.NewReader(buf))
	handler := NewTcpHandler("")
	protocol, ack, err := handler.attemptDeviceLogin(reader)

	assert.NoError(t, err, "device login should succeed")
	assert.IsType(t, &wanway.WanwayProtocol{}, protocol, "protocol should be of type TeltonikaProtocol")
	assert.Equal(t, "357454075177072", protocol.GetDeviceIdentifier(), "imei should be parsed correctly")
	// assert.Equal(t, []byte{0x78, 0x78, 0x11, 0x01, 0x00, 0x01, 0xbb, 0x1a, 0x0d, 0x0a}, ack, "login ack should be of the format as wanway expects")
	assert.Equal(t, []byte{0x78, 0x78, 0x11, 0x01, 0x00, 0x01, 0x91, 0x40, 0x0d, 0x0a}, ack, "login ack should be of the format as wanway expects")
}

func TestUnkonwnDeviceLogin(t *testing.T) {
	buf, _ := hex.DecodeString("7676fafafafa")
	reader := bufio.NewReader(bytes.NewReader(buf))
	handler := NewTcpHandler("")
	protocol, ack, err := handler.attemptDeviceLogin(reader)

	assert.Nil(t, protocol, "protocol should be nil")
	assert.Nil(t, ack, "ack should be nil")
	assert.ErrorIs(t, err, errs.ErrUnknownDeviceType, "error should be ErrUnknownDevice")
}
