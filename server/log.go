package server

import (
	"fmt"
	"io"

	"github.com/reeflective/team/internal/log"
	"github.com/sirupsen/logrus"
)

// NamedLogger returns a new logging "thread" which should grossly
// indicate the package/general domain, and a more precise flow/stream.
func (ts *Server) NamedLogger(pkg, stream string) *logrus.Entry {
	return ts.log().WithFields(logrus.Fields{
		log.PackageFieldKey: pkg,
		"stream":            stream,
	})
}

// SetLogLevel is a utility to change the logging level of the stdout logger.
func (ts *Server) SetLogLevel(level int) {
	if ts.stdoutLogger == nil {
		return
	}

	if uint32(level) > uint32(logrus.TraceLevel) {
		level = int(logrus.TraceLevel)
	}

	ts.stdoutLogger.SetLevel(logrus.Level(uint32(level)))
}

// WithLoggerStdout sets the source to which the stdout logger (not any file logger) should write to.
// This option is used by the teamserver/teamclient cobra command tree to coordinate its basic I/O/err.
func (ts *Server) SetLogWriter(stdout, stderr io.Writer) {
	// TODO: Maybe if the writer was already set in options, ignore ?
	ts.stdoutLogger.Out = stdout
	// TODO: Pass stderr to log internals.
}

// Initialize loggers in files/stdout according to options.
func (ts *Server) initLogging() (err error) {
	// No logging means only stdout with warn level
	if ts.opts.noLogs || ts.opts.noFiles {
		ts.stdoutLogger = log.NewStdio(logrus.WarnLevel)
		return nil
	}

	// Ensure all teamserver-specific directories are writable.
	if err := ts.checkWritableFiles(); err != nil {
		return fmt.Errorf("%w: %w", ErrDirectory, err)
	}

	// If user supplied a logger, use it in place of the
	// stdout logger, since the file logger is optional.
	if ts.opts.logger != nil {
		ts.stdoutLogger = ts.opts.logger
	}

	level := logrus.Level(ts.opts.config.Log.Level)

	// Either use default logfile or user-specified one.
	ts.fileLogger, ts.stdoutLogger, err = log.NewClient(ts.opts.logFile, level)
	if err != nil {
		return err
	}

	return nil
}

// log returns a non-nil logger for the server:
// if file logging is disabled, it returns the stdout-only logger,
// otherwise returns the file logger equipped with a stdout hook.
func (ts *Server) log() *logrus.Logger {
	if ts.fileLogger == nil {
		return ts.stdoutLogger
	}

	return ts.fileLogger
}

func (ts *Server) errorf(msg string, format ...any) error {
	logged := fmt.Errorf(msg, format...)
	ts.log().Error(logged)

	return logged
}

func (ts *Server) errorWith(log *logrus.Entry, msg string, format ...any) error {
	logged := fmt.Errorf(msg, format...)

	if log != nil {
		log.Error(logged)
	} else {
		ts.log().Error(logged)
	}

	return logged
}