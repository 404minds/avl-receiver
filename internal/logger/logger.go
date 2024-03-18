package logger

import (
	"go.uber.org/zap"
)

// Logger is a wrapper around zap.Logger
// we can configure it as we want
func zapLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment(
	//zap.AddStacktrace(zap.PanicLevel)
	)
	return logger
}

var Logger = zapLogger()
