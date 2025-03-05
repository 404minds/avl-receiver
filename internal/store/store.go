package store

import (
	"context"
	"github.com/404minds/avl-receiver/internal/types"
)

type Store interface {
	Process(ctx context.Context)
	Response(ctx context.Context)
	GetProcessChan() chan *types.DeviceStatus
	GetResponseChan() chan *types.DeviceResponse
	GetCloseChan() chan bool
	GetCloseResponseChan() chan bool
}
