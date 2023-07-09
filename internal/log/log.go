package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// NewStdout returns a logger printing its results to stdout.
// Default logging level: error (no output when things work)
func NewStdout(app string, level logrus.Level) *logrus.Logger {
	stdLogger := logrus.New()
	stdLogger.Formatter = &textFormatter{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}

	stdLogger.SetLevel(logrus.WarnLevel)
	stdLogger.SetReportCaller(true)
	stdLogger.Out = os.Stdout

	return stdLogger
}

// NewClient creates a default in-memory logger which prints everything out (with formatting)
// to os.Stdout, and a side-hook writing the log event in a slightly different format to a file.
// Logging levels as independent between stdout and text file.
func NewClient(path string, app string, level logrus.Level) (*logrus.Logger, error) {
	txtLogger := logrus.New()
	txtLogger.Formatter = &textFormatter{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}

	txtLogger.SetLevel(logrus.InfoLevel)
	txtLogger.SetReportCaller(true)

	// Output both to the screen and to a file.
	txtLogger.Out = io.Discard
	txtLogger.AddHook(newTxtHook(path, app, level, txtLogger))

	return txtLogger, nil
}

// NewRoot returns a logger writing to the central log file of the teamserver, JSON-encoded.
func NewRoot(app, logDir string, level logrus.Level) (*logrus.Logger, error) {
	rootLogger := logrus.New()
	rootLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := filepath.Join(logDir, fmt.Sprintf("%s.json", app))
	jsonFile, err := os.OpenFile(jsonFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("Failed to open log file %v", err)
	}
	rootLogger.Out = jsonFile
	rootLogger.SetLevel(logrus.WarnLevel)
	rootLogger.SetReportCaller(true)
	rootLogger.AddHook(newTxtHook(logDir, app, level, rootLogger))
	return rootLogger, nil
}

// NewAudit returns a new client gRPC connections audit logger, JSON-encoded.
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

// newText returns a new logger writing to a given file.
func newText(path, name string) (*logrus.Logger, error) {
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
	txtLogger.SetLevel(logrus.ErrorLevel)
	return txtLogger, nil
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
