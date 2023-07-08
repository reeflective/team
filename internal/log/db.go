package log

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
)

type gormWriter struct {
	log *logrus.Entry
}

func (w gormWriter) Printf(format string, args ...interface{}) {
	w.log.Printf(format, args...)
}

// NewDatabase returns a logger suitable as logrus database logging middleware.
func NewDatabase(log *logrus.Entry, level string) logger.Interface {
	logConfig := logger.Config{
		SlowThreshold: time.Second,
		Colorful:      true,
		LogLevel:      logger.Info,
	}
	switch strings.ToLower(level) {
	case "silent":
		logConfig.LogLevel = logger.Silent
	case "err":
		fallthrough
	case "error":
		logConfig.LogLevel = logger.Error
	case "warning":
		fallthrough
	case "warn":
		logConfig.LogLevel = logger.Warn
	case "info":
		fallthrough
	default:
		logConfig.LogLevel = logger.Info
	}

	return logger.New(gormWriter{log: log}, logConfig)
}
