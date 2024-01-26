package store

type Store interface {
	Process() error
	GetProcessChan() chan interface{}
	GetCloseChan() chan bool
}