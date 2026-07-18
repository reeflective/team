package server

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
	"log/slog"
	"path/filepath"

	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/log"
)

// NamedLogger returns a new logger tagged with a package/domain and a more
// precise flow/stream. The events are logged according to the teamserver
// logging backend setup.
func (ts *Server) NamedLogger(pkg, stream string) *slog.Logger {
	return ts.logger.Named(pkg, stream)
}

// SetLogLevel sets the logging level of teamserver loggers (excluding audit ones).
func (ts *Server) SetLogLevel(level int) {
	if ts.logger == nil {
		return
	}

	ts.logger.SetLevel(log.LevelFrom(level))
}

// AuditLogger returns a special logger writing its event entries to an audit
// log file (default audit.json), distinct from other teamserver log files.
// Handler implementations will want to use this for logging various teamclient
// application requests, with this logger used somewhere in your handler middleware.
func (ts *Server) AuditLogger() (*slog.Logger, error) {
	if ts.opts.inMemory || ts.opts.noLogs {
		return ts.log(), nil
	}

	// Open the audit file and wrap it in a JSON audit logger.
	auditPath := filepath.Join(ts.LogsDir(), "audit.json")

	auditFile, err := ts.fs.OpenFile(auditPath, assets.FileWriteOpenMode, assets.FileWritePerm)
	if err != nil {
		return nil, ts.errorf("%w: %w", ErrLogging, err)
	}

	return log.NewAudit(auditFile), nil
}

// Initialize loggers in files/stdout according to options.
func (ts *Server) initLogging() (err error) {
	// If the user supplied a logging handler, it becomes the sole backend.
	if ts.opts.logger != nil {
		ts.logger = log.NewFromHandler(ts.opts.logger)
		return nil
	}

	logFile := filepath.Join(ts.LogsDir(), log.FileName(ts.Name(), true))

	// If the teamserver should log to a given file.
	if ts.opts.logFile != "" {
		logFile = ts.opts.logFile
	}

	fileLevel := log.LevelFrom(ts.opts.config.Log.Level)

	// Open the log file (on disk or in the in-memory filesystem).
	logfile, err := ts.fs.OpenFile(logFile, assets.FileWriteOpenMode, assets.FileWritePerm)
	if err != nil {
		return err
	}

	// Console logs warnings and above by default, the file logs at the
	// configured level; the console can be restyled via WithConsoleOptions.
	ts.logger = log.New(logfile, slog.LevelWarn, fileLevel, ts.opts.consoleStyle)

	return nil
}

// log returns the underlying, non-nil *slog.Logger for the server.
func (ts *Server) log() *slog.Logger {
	if ts.logger == nil {
		ts.logger = log.NewStdio(slog.LevelWarn)
	}

	return ts.logger.Logger()
}

func (ts *Server) errorf(msg string, format ...any) error {
	logged := fmt.Errorf(msg, format...)
	ts.log().Error(logged.Error())

	return logged
}

func (ts *Server) errorWith(logger *slog.Logger, msg string, format ...any) error {
	logged := fmt.Errorf(msg, format...)

	if logger != nil {
		logger.Error(logged.Error())
	} else {
		ts.log().Error(logged.Error())
	}

	return logged
}
