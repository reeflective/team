package client

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/reeflective/team/internal/log"
	"github.com/sirupsen/logrus"
)

// NamedLogger returns a new logging "thread" which should grossly
// indicate the package/general domain, and a more precise flow/stream.
func (tc *Client) NamedLogger(pkg, stream string) *logrus.Entry {
	return tc.log().WithFields(logrus.Fields{
		log.PackageFieldKey: pkg,
		"stream":            stream,
	})
}

// WithLoggerStdout sets the source to which the stdout logger (not any file logger) should write to.
// This option is used by the teamserver/teamclient cobra command tree to coordinate its basic I/O/err.
func (tc *Client) SetLogWriter(stdout, stderr io.Writer) {
	tc.stdoutLogger.Out = stdout
	// TODO: Pass stderr to log internals.
}

// SetLogLevel is a utility to change the logging level of the stdout logger.
func (tc *Client) SetLogLevel(level int) {
	if tc.stdoutLogger == nil {
		return
	}

	if uint32(level) > uint32(logrus.TraceLevel) {
		level = int(logrus.TraceLevel)
	}

	tc.stdoutLogger.SetLevel(logrus.Level(uint32(level)))

	if tc.fileLogger != nil {
		tc.fileLogger.SetLevel(logrus.Level(uint32(level)))
	}
}

// Initialize loggers in files/stdout according to options.
func (tc *Client) initLogging() (err error) {
	// By default, the stdout logger is never nil.
	// We might overwrite it below if using our defaults.
	tc.stdoutLogger = log.NewStdio(logrus.WarnLevel)

	// Path to our client log file, and open it (in mem or on disk)
	logFile := filepath.Join(tc.LogsDir(), log.FileName(tc.Name(), false))

	// If the teamclient should log to a predefined file.
	if tc.opts.logFile != "" {
		logFile = tc.opts.logFile
	}

	// If user supplied a logger, use it in place of the
	// file-based logger, since the file logger is optional.
	if tc.opts.logger != nil {
		tc.fileLogger = tc.opts.logger
		return nil
	}

	// Create the loggers writing to this file, and hooked to write to stdout as well.
	tc.fileLogger, tc.stdoutLogger, err = log.Init(tc.fs, logFile, logrus.InfoLevel)
	if err != nil {
		return err
	}

	return nil
}

// log returns a non-nil logger for the client:
// if file logging is disabled, it returns the stdout-only logger,
// otherwise returns the file logger equipped with a stdout hook.
func (tc *Client) log() *logrus.Logger {
	if tc.fileLogger != nil {
		return tc.fileLogger
	}

	if tc.stdoutLogger == nil {
		tc.stdoutLogger = log.NewStdio(logrus.WarnLevel)
	}

	return tc.stdoutLogger
}

func (tc *Client) errorf(msg string, format ...any) error {
	logged := fmt.Errorf(msg, format...)
	tc.log().Error(logged)

	return logged
}
