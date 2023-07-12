package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/reeflective/team/internal/assets"
	"github.com/sirupsen/logrus"
)

const (
	FileReadPerm  = 0o600 // FileReadPerm is the permission bit given to the OS when reading files.
	DirPerm       = 0o700 // DirPerm is the permission bit given to teamserver/client directories.
	FileWritePerm = 0o644 // FileWritePerm is the permission bit given to the OS when writing files.

	FileWriteOpenMode = os.O_APPEND | os.O_CREATE | os.O_WRONLY // Opening log files in append/create/write-only mode.

	ClientLogFileExt = "teamclient.log" // Log files of all teamclients have this extension by default.
	ServerLogFileExt = "teamserver.log" // Log files of all teamserver have this extension by default.
)

func Init(fs *assets.FS, file string, level logrus.Level) (*logrus.Logger, *logrus.Logger, error) {
	logFile, err := fs.OpenFile(file, FileWriteOpenMode, FileWritePerm)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to open log file %w", err)
	}

	// Text-format logger, writing to file.
	fileLog := logrus.New()
	fileLog.Formatter = &stdoutHook{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}
	fileLog.Out = io.Discard

	fileLog.SetLevel(logrus.InfoLevel)
	fileLog.SetReportCaller(true)

	// File output
	fileLog.AddHook(newTxtHook(logFile, level, fileLog))

	// Stdout/err output, with special formatting.
	stdioHook := newStdioHook()
	fileLog.AddHook(stdioHook)

	return fileLog, stdioHook.logger, nil
}

// NewStdio returns a logger configured to output its events to the system stdio:
// - Info/Debug/Trace logs are written to os.Stdout.
// - Warn/Error/Fatal/Panic are written to os.Stderr.
func NewStdio(level logrus.Level) *logrus.Logger {
	stdLogger := logrus.New()
	stdLogger.Formatter = &stdoutHook{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}

	stdLogger.SetLevel(level)
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

// NewJSON returns a logger writing to the central log file of the teamserver, JSON-encoded.
func NewJSON(fs *assets.FS, file string, level logrus.Level) (*logrus.Logger, error) {
	rootLogger := logrus.New()
	rootLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := fmt.Sprintf("%s.json", file)

	logFile, err := fs.OpenFile(jsonFilePath, FileWriteOpenMode, FileWritePerm)
	if err != nil {
		return nil, fmt.Errorf("Failed to open log file %w", err)
	}

	rootLogger.Out = logFile
	rootLogger.SetLevel(logrus.InfoLevel)
	rootLogger.SetReportCaller(true)
	rootLogger.AddHook(newTxtHook(logFile, level, rootLogger))

	return rootLogger, nil
}

// NewAudit returns a new client gRPC connections audit logger, JSON-encoded.
func NewAudit(fs *assets.FS, logDir string) (*logrus.Logger, error) {
	auditLogger := logrus.New()
	auditLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := filepath.Join(logDir, "audit.json")

	logFile, err := fs.OpenFile(jsonFilePath, FileWriteOpenMode, FileWritePerm)
	if err != nil {
		return nil, fmt.Errorf("Failed to open log file %w", err)
	}

	auditLogger.Out = logFile
	auditLogger.SetLevel(logrus.DebugLevel)

	return auditLogger, nil
}

// NewText returns a new logger writing to a given file.
func NewText(file io.Writer) (*logrus.Logger, error) {
	txtLogger := logrus.New()
	txtLogger.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}

	txtLogger.Out = file
	txtLogger.SetLevel(logrus.InfoLevel)

	return txtLogger, nil
}

// LevelFrom - returns level from int.
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

func FileName(name string, server bool) string {
	if server {
		return fmt.Sprintf("%s.%s", name, ServerLogFileExt)
	}
	return fmt.Sprintf("%s.%s", name, ClientLogFileExt)
}
