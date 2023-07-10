package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func NewStdio(level logrus.Level) *logrus.Logger {
	stdLogger := logrus.New()
	stdLogger.Formatter = &stdoutHook{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}

	stdLogger.SetLevel(logrus.WarnLevel)
	stdLogger.SetReportCaller(true)
	stdLogger.Out = io.Discard

	// Info/debug/trace is given to a stdout logger.
	stdoutHook := newLoggerStdout()
	stdLogger.AddHook(stdoutHook)

	// Warn/error/panics/fatals are given to stderr.
	stderrHook := newLoggerStderr()
	stdLogger.AddHook(stderrHook)

	return stdLogger
}

// NewClient creates a default in-memory logger which prints everything out (with formatting)
// to os.Stdout, and a side-hook writing the log event in a slightly different format to a file.
// Logging levels as independent between stdout and text file.
func NewClient(logfile string, level logrus.Level) (file, stdout *logrus.Logger, err error) {
	txtLogger := logrus.New()
	txtLogger.Formatter = &stdoutHook{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}
	txtLogger.Out = io.Discard

	txtLogger.SetLevel(logrus.InfoLevel)
	txtLogger.SetReportCaller(true)

	// File output
	txtLogger.AddHook(newTxtHook(logfile, level, txtLogger))

	// Stdio
	stdioHook := newStdioHook()
	txtLogger.AddHook(stdioHook)

	return txtLogger, stdioHook.logger, nil
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
	rootLogger.SetLevel(logrus.InfoLevel)
	rootLogger.SetReportCaller(true)
	rootLogger.AddHook(newTxtHook(logDir, level, rootLogger))
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

// NewText returns a new logger writing to a given file.
func NewText(logFile string) (*logrus.Logger, error) {
	txtLogger := logrus.New()
	txtLogger.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}
	txtFile, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("Failed to open log file %v", err)
	}

	txtLogger.Out = txtFile
	txtLogger.SetLevel(logrus.InfoLevel)

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
