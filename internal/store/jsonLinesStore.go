package store

import (
	"encoding/json"
	"fmt"
	"os"

	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"github.com/404minds/avl-receiver/internal/types"
	"go.uber.org/zap"
)

var logger = configuredLogger.Logger

type JsonLinesStore struct {
	File        *os.File
	ProcessChan chan types.DeviceStatus
	//ResponseChan chan types.DeviceResponse
	CloseChan chan bool
	DeviceID  string
}

//func (s *JsonLinesStore) GetResponseChan() chan types.DeviceResponse {
//	return s.ResponseChan
//}

func (s *JsonLinesStore) GetProcessChan() chan types.DeviceStatus {
	return s.ProcessChan
}

func (s *JsonLinesStore) GetCloseChan() chan bool {
	return s.CloseChan
}

func (s *JsonLinesStore) Process() {
	for {
		select {
		case data := <-s.ProcessChan:
			b, err := json.Marshal(data)
			if err != nil {
				logger.Error("failed to write record to file", zap.String("deviceId", s.DeviceID), zap.Error(err))
			}
			fmt.Fprintln(s.File, string(b))
			s.File.Sync()
		case <-s.CloseChan:
			return
		}
	}
}

//func (s *JsonLinesStore) Response() {
//	for {
//		select {
//		case data := <-s.ResponseChan:
//			b, err := json.Marshal(data)
//			if err != nil {
//				logger.Error("failed to write record to file", zap.String("deviceId", s.DeviceID), zap.Error(err))
//			}
//			fmt.Fprintln(s.File, string(b))
//			s.File.Sync()
//		case <-s.CloseChan:
//			return
//		}
//	}
//}
