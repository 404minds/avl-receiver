package store

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/404minds/avl-receiver/internal/devices/teltonika"
	configuredLogger "github.com/404minds/avl-receiver/internal/logger"
	"go.uber.org/zap"
)

var logger = configuredLogger.Logger

type JsonLinesStore struct {
	File        *os.File
	ProcessChan chan interface{}
	CloseChan   chan bool
}

func (s *JsonLinesStore) GetProcessChan() chan interface{} {
	return s.ProcessChan
}

func (s *JsonLinesStore) GetCloseChan() chan bool {
	return s.CloseChan
}

func (s *JsonLinesStore) Process() error {
	for {
		select {
		case data := <-s.ProcessChan:
			switch record := data.(type) {
			case teltonika.TeltonikaRecord:
				b, err := json.Marshal(record)
				if err != nil {
					logger.Error("failed to write Teltonika record to file", zap.String("imei", record.IMEI))
					logger.Error(err.Error())
				}
				fmt.Fprintln(s.File, string(b))
				s.File.Sync()
			default:
				logger.Error("invalid data type received to write to file")
			}
		case <-s.CloseChan:
			return nil
		}
	}
}
