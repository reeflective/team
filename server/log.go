package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// NamedLogger - Returns a logger wrapped with pkg/stream fields
func (s *Server) NamedLogger(pkg, stream string) *logrus.Entry {
	return s.log.WithFields(logrus.Fields{
		"pkg":    pkg,
		"stream": stream,
	})
}

// AuditLogger - Single audit log
// var AuditLogger = newAuditLogger()

func (s *Server) newAuditLogger() *logrus.Logger {
	auditLogger := logrus.New()
	auditLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := filepath.Join(s.LogsDir(), "audit.json")
	jsonFile, err := os.OpenFile(jsonFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file %v", err))
	}
	auditLogger.Out = jsonFile
	auditLogger.SetLevel(logrus.DebugLevel)
	return auditLogger
}

// RootLogger - Returns the root logger
func (s *Server) rootLogger() *logrus.Logger {
	rootLogger := logrus.New()
	rootLogger.Formatter = &logrus.JSONFormatter{}
	jsonFilePath := filepath.Join(s.LogsDir(), fmt.Sprintf("%s.json", s.name))
	jsonFile, err := os.OpenFile(jsonFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file %v", err))
	}
	rootLogger.Out = jsonFile
	rootLogger.SetLevel(logrus.DebugLevel)
	rootLogger.SetReportCaller(true)
	rootLogger.AddHook(s.NewTxtHook("root"))
	return rootLogger
}

// RootLogger - Returns the root logger
func (s *Server) txtLogger() *logrus.Logger {
	txtLogger := logrus.New()
	txtLogger.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}
	txtFilePath := filepath.Join(s.LogsDir(), "server.log")
	txtFile, err := os.OpenFile(txtFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file %v", err))
	}
	txtLogger.Out = txtFile
	txtLogger.SetLevel(logrus.DebugLevel)
	return txtLogger
}

// TxtHook - Hook in a textual version of the logs
type TxtHook struct {
	Name   string
	app    string
	logger *logrus.Logger
}

// NewTxtHook - returns a new txt hook
func (s *Server) NewTxtHook(name string) *TxtHook {
	hook := &TxtHook{
		Name:   name,
		app:    s.name,
		logger: s.txtLogger(),
	}
	return hook
}

// Fire - Implements the fire method of the Logrus hook
func (hook *TxtHook) Fire(entry *logrus.Entry) error {
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
func (hook *TxtHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// RootLogger - Returns the root logger
func (s *Server) stdoutLogger() *logrus.Logger {
	txtLogger := logrus.New()
	txtLogger.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}
	txtLogger.Out = os.Stdout
	txtLogger.SetLevel(logrus.DebugLevel)
	return txtLogger
}

// TxtHook - Hook in a textual version of the logs
type StdoutHook struct {
	Name   string
	app    string
	logger *logrus.Logger
}

// NewTxtHook - returns a new txt hook
func (s *Server) newStdoutHook(name string) *StdoutHook {
	hook := &StdoutHook{
		Name:   name,
		app:    s.name,
		logger: s.stdoutLogger(),
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

// levelFrom - returns level from int
func levelFrom(level int) logrus.Level {
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
