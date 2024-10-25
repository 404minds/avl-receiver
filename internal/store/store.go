package store

import "github.com/404minds/avl-receiver/internal/types"

type Store interface {
	Process()
	//Response()

	GetProcessChan() chan types.DeviceStatus
	//GetResponseChan() chan types.DeviceResponse

	GetCloseChan() chan bool
}
