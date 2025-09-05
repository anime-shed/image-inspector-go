package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

func init() {
	Logger = logrus.New()

	// Set output to stdout
	Logger.SetOutput(os.Stdout)

	// Set log level from environment or default to Info
	level := os.Getenv("LOG_LEVEL")
	switch level {
	case "debug":
		Logger.SetLevel(logrus.DebugLevel)
	case "warn":
		Logger.SetLevel(logrus.WarnLevel)
	case "error":
		Logger.SetLevel(logrus.ErrorLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}

	// Set JSON formatter for structured logging
	Logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
}

// WithFields creates a new entry with the given fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Logger.WithFields(fields)
}

// WithField creates a new entry with a single field
func WithField(key string, value interface{}) *logrus.Entry {
	return Logger.WithField(key, value)
}

// WithError creates a new entry with an error field
func WithError(err error) *logrus.Entry {
	return Logger.WithError(err)
}

// Info logs an info message
func Info(msg string) {
	Logger.Info(msg)
}

// Error logs an error message
func Error(msg string) {
	Logger.Error(msg)
}

// Debug logs a debug message
func Debug(msg string) {
	Logger.Debug(msg)
}

// Warn logs a warning message
func Warn(msg string) {
	Logger.Warn(msg)
}
