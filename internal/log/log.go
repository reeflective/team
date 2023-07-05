package log

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// NewLoggerStream creates a default in-memory logger which
// prints everything out (with formatting) to os.Stdout.
// All clients and servers make use of this logger.
func NewLoggerStream() *logrus.Logger {
	txtLogger := logrus.New()
	txtLogger.Formatter = &CustomFormatter{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}

	txtLogger.Out = os.Stdout
	txtLogger.SetLevel(logrus.InfoLevel)

	return txtLogger
}

// NamedLogger - Returns a logger wrapped with package/stream fields
func NamedLogger(log *logrus.Logger, pkg, stream string) *logrus.Entry {
	return log.WithFields(logrus.Fields{
		PackageFieldKey: pkg,
		"stream":        stream,
	})
}

// NewLoggerRoot returns the root logger.
func NewLoggerRoot(app, name, logDir string) (*logrus.Logger, error) {
	rootLogger := logrus.New()
	rootLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := filepath.Join(logDir, fmt.Sprintf("%s.json", app))
	jsonFile, err := os.OpenFile(jsonFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("Failed to open log file %v", err)
	}
	rootLogger.Out = jsonFile
	rootLogger.SetLevel(logrus.DebugLevel)
	rootLogger.SetReportCaller(true)
	rootLogger.AddHook(NewTxtHook(app, name, logDir))
	return rootLogger, nil
}

// NewLoggerText returns a new logger writing to a given file.
func NewLoggerText(logDir string) (*logrus.Logger, error) {
	txtLogger := logrus.New()
	txtLogger.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}
	txtFilePath := filepath.Join(logDir, "server.log")
	txtFile, err := os.OpenFile(txtFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("Failed to open log file %v", err)
	}
	txtLogger.Out = txtFile
	txtLogger.SetLevel(logrus.DebugLevel)
	return txtLogger, nil
}

// NewLoggerAudit returns a new client gRPC connections audit logger.
func NewLoggerAudit(logDir string) (*logrus.Logger, error) {
	auditLogger := logrus.New()
	auditLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := filepath.Join(logDir, "audit.json")
	jsonFile, err := os.OpenFile(jsonFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("Failed to open log file %v", err)
	}
	auditLogger.Out = jsonFile
	auditLogger.SetLevel(logrus.DebugLevel)
	return auditLogger, nil
}

// TxtHook - Hook in a textual version of the logs
type StdoutHook struct {
	Name   string
	app    string
	logger *logrus.Logger
}

// NewTxtHook - returns a new txt hook
func NewStdoutHook(app, name, logDir string) *StdoutHook {
	hook := &StdoutHook{
		Name:   name,
		app:    app,
		logger: NewLoggerStream(),
	}
	return hook
}

// Fire - Implements the fire method of the Logrus hook
func (hook *StdoutHook) Fire(entry *logrus.Entry) error {
	if hook.logger == nil {
		return errors.New("no txt logger")
	}

	// Determine the caller (filename/line number)
	srcFile := "<no caller>"
	if entry.HasCaller() {
		wiregostIndex := strings.Index(entry.Caller.File, hook.app)
		srcFile = entry.Caller.File
		if wiregostIndex != -1 {
			srcFile = srcFile[wiregostIndex:]
		}
	}

	switch entry.Level {
	case logrus.PanicLevel:
		hook.logger.Panicf("[%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.FatalLevel:
		hook.logger.Fatalf("[%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.ErrorLevel:
		hook.logger.Errorf("[%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.WarnLevel:
		hook.logger.Warnf("[%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.InfoLevel:
		hook.logger.Infof("[%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.DebugLevel, logrus.TraceLevel:
		hook.logger.Debugf("[%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	}

	return nil
}

// Levels - Hook all levels
func (hook *StdoutHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// LevelFrom - returns level from int
func LevelFrom(level int) logrus.Level {
	switch level {
	case 0:
		return logrus.PanicLevel
	case 1:
		return logrus.FatalLevel
	case 2:
		return logrus.ErrorLevel
	case 3:
		return logrus.WarnLevel
	case 4:
		return logrus.InfoLevel
	case 5:
		return logrus.DebugLevel
	case 6:
		return logrus.TraceLevel
	}
	return logrus.DebugLevel
}

// Initialize logging
// func (c *Client) initLogging(appDir string) *os.File {
// 	logFile, err := os.OpenFile(path.Join(appDir, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
// 	if err != nil {
// 		panic(fmt.Sprintf("[!] Error opening file: %s", err))
// 	}
//
// 	// Initialize the log handler
// 	jsonOptions := &slog.HandlerOptions{
// 		Level: slog.LevelDebug,
// 	}
//
// 	jsonHandler := slog.NewJSONHandler(logFile, jsonOptions)
// 	c.log = slog.New(jsonHandler)
//
// 	return logFile
// }
