package log

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// NamedLogger - Returns a logger wrapped with package/stream fields
func NamedLogger(log *logrus.Logger, pkg, stream string) *logrus.Entry {
	return log.WithFields(logrus.Fields{
		PackageFieldKey: pkg,
		"stream":        stream,
	})
}

// NewClient creates a default in-memory logger which
// prints everything out (with formatting) to os.Stdout.
// All clients and servers make use of this logger.
func NewClient(path string, app string) (*logrus.Logger, error) {
	txtLogger := logrus.New()
	txtLogger.Formatter = &textFormatter{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}

	txtLogger.SetLevel(logrus.DebugLevel)
	txtLogger.SetReportCaller(true)

	// Output both to the screen and to a file.
	txtLogger.Out = os.Stdout
	txtLogger.AddHook(newTxtHook(path, app))

	return txtLogger, nil
}

// NewRoot returns the root logger.
func NewRoot(app, logDir string) (*logrus.Logger, error) {
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
	rootLogger.AddHook(newTxtHook(logDir, app))
	return rootLogger, nil
}

// NewText returns a new logger writing to a given file.
func NewText(path, name string) (*logrus.Logger, error) {
	txtLogger := logrus.New()
	txtLogger.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}
	txtFilePath := filepath.Join(path, fmt.Sprintf("%s.log", name))
	txtFile, err := os.OpenFile(txtFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("Failed to open log file %v", err)
	}
	txtLogger.Out = txtFile
	txtLogger.SetLevel(logrus.DebugLevel)
	return txtLogger, nil
}

// NewAudit returns a new client gRPC connections audit logger.
func NewAudit(logDir string) (*logrus.Logger, error) {
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
