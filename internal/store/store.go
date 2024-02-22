package store

import "github.com/404minds/avl-receiver/internal/types"

type Store interface {
	Process() error
	GetProcessChan() chan types.DeviceStatus
	GetCloseChan() chan bool
}
