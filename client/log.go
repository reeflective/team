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
	"log/slog"
	"path/filepath"

	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/log"
)

// NamedLogger returns a new logger tagged with a package/domain and a more
// precise flow/stream. The events are logged according to the teamclient
// logging backend setup.
func (tc *Client) NamedLogger(pkg, stream string) *slog.Logger {
	return tc.logger.Named(pkg, stream)
}

// SetLogWriter sets the streams to which the console logger (not the file logger)
// should write. This is used by the teamclient cobra command tree to synchronize
// its stdout/stderr with the teamclient backend.
func (tc *Client) SetLogWriter(stdout, stderr io.Writer) {
	tc.logger.SetOutput(stdout, stderr)
}

// SetLogLevel sets the logging level of all teamclient loggers.
func (tc *Client) SetLogLevel(level int) {
	if tc.logger == nil {
		return
	}

	tc.logger.SetLevel(log.LevelFrom(level))
}

// log returns the underlying, non-nil *slog.Logger for the client.
func (tc *Client) log() *slog.Logger {
	if tc.logger == nil {
		tc.logger = log.NewStdio(slog.LevelWarn)
	}

	return tc.logger.Logger()
}

func (tc *Client) errorf(msg string, format ...any) error {
	logged := fmt.Errorf(msg, format...)
	tc.log().Error(logged.Error())

	return logged
}

func (tc *Client) initLogging() (err error) {
	// If the user supplied a logging handler, it becomes the sole backend.
	if tc.opts.logger != nil {
		tc.logger = log.NewFromHandler(tc.opts.logger)
		return nil
	}

	// Path to our client log file, and open it (in mem or on disk).
	logFile := filepath.Join(tc.LogsDir(), log.FileName(tc.Name(), false))

	// If the teamclient should log to a predefined file.
	if tc.opts.logFile != "" {
		logFile = tc.opts.logFile
	}

	// Open the log file (on disk or in the in-memory filesystem).
	logfile, err := tc.fs.OpenFile(logFile, assets.FileWriteOpenMode, assets.FileWritePerm)
	if err != nil {
		return err
	}

	// Console logs warnings and above by default, the file logs from info;
	// the console can be restyled via WithConsoleOptions.
	tc.logger = log.New(logfile, slog.LevelWarn, slog.LevelInfo, tc.opts.consoleStyle)

	// Apply the configured console format (console/text/json), if any.
	if tc.opts.logFormat != "" {
		tc.logger.SetLogFormat(tc.opts.logFormat)
	}

	return nil
}

// SetLogFormat rebuilds the teamclient console stream in the given format
// (log.FormatConsole/Text/JSON). The file logger stays plain text. Intended for
// use at startup (eg. from the teamclient `--log-format` flag).
func (tc *Client) SetLogFormat(format log.Format) {
	if tc.logger == nil {
		return
	}

	tc.logger.SetLogFormat(format)
}
