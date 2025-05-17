package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// LogLevel represents the logging level
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
)

// Init initializes the logger with the specified configuration
func Init(development bool, level LogLevel) error {
	var err error
	var config zap.Config

	if development {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	// Set the log level based on the provided level
	switch level {
	case DebugLevel:
		config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case InfoLevel:
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case WarnLevel:
		config.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case ErrorLevel:
		config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	log, err = config.Build()
	return err
}

// Get returns the logger instance
func Get() *zap.Logger {
	return log
}

// Sync flushes any buffered log entries
func Sync() error {
	return log.Sync()
}
