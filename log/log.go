package log

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
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

const (
	// ClientLogFileExt is used as extension by all main teamclients log files by default.
	ClientLogFileExt = "teamclient.log"
	// ServerLogFileExt is used as extension by all teamservers core log files by default.
	ServerLogFileExt = "teamserver.log"
)

// Logger is the unified logging backend shared by teamclients and teamservers.
// It wraps a single *slog.Logger whose handler, by default, fans out log records
// to a colored console (with a stdout/stderr level split) and to a text log file,
// each with its own runtime-adjustable level. It can also wrap a single
// user-provided slog.Handler, in which case the console/file split and the
// level knobs do not apply.
//
// It is exported so that consumers running the core in-memory (server.WithInMemory
// / client.WithInMemory) can build their OWN ephemeral logger: open a file on the
// filesystem returned by Server.Filesystem() / Client.Filesystem() and pass it as
// the io.Writer to New.
type Logger struct {
	logger *slog.Logger
	stdio  *slog.LevelVar // console level (nil when a custom handler is used)
	file   *slog.LevelVar // file level (nil when a custom handler is used)
	out    *swapWriter    // console stdout (nil when a custom handler is used)
	err    *swapWriter    // console stderr (nil when a custom handler is used)

	// Retained so the stdio format can be (re)built, eg. by SetLogFormat.
	logFile io.Writer
	style   func(*ConsoleOptions)
	format  Format
}

// swapWriter is an io.Writer whose destination can be swapped at runtime and is
// safe for concurrent use. It lets a live logger redirect its console streams
// (eg. onto a cobra command's stdout/stderr) without rebuilding handlers, and is
// shared by reference across handlers cloned via WithAttrs/WithGroup.
type swapWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func newSwapWriter(w io.Writer) *swapWriter {
	return &swapWriter{w: w}
}

func (s *swapWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.w.Write(p)
}

func (s *swapWriter) Set(w io.Writer) {
	if w == nil {
		return
	}

	s.mu.Lock()
	s.w = w
	s.mu.Unlock()
}

// New builds the default teamserver/teamclient logger: a colored console logger
// (info/debug/trace to stdout, warn and above to stderr), tee'd with a plain-text
// log file when logFile is non-nil.
//
// logFile is any io.Writer — the caller opens it however it wants (os.OpenFile, an
// afero file for in-memory use, a buffer...). Pass nil for a console-only logger.
//
// The optional style callback restyles the console/file columns (level markers,
// colors, timestamp, widths) before the handlers are built; it is applied on top
// of the built-in defaults (timestamps on), so a nil callback yields the default
// look. The library controls the level/output fields itself.
func New(logFile io.Writer, stdioLevel, fileLevel slog.Level, style func(*ConsoleOptions)) *Logger {
	stdioVar := &slog.LevelVar{}
	stdioVar.Set(stdioLevel)

	logger := &Logger{
		stdio:   stdioVar,
		out:     newSwapWriter(os.Stdout),
		err:     newSwapWriter(os.Stderr),
		logFile: logFile,
		style:   style,
		format:  FormatConsole,
	}

	if logFile != nil {
		fileVar := &slog.LevelVar{}
		fileVar.Set(fileLevel)
		logger.file = fileVar
	}

	logger.build()

	return logger
}

// build (re)creates the underlying slog.Logger from the retained inputs: the
// stdio handler for the current format, tee'd with the plain-text file handler
// when a log file is set. It reuses the level vars and swap writers, so SetLevel
// and SetOutput keep working across rebuilds.
func (l *Logger) build() {
	stdio := l.stdioHandler()

	if l.logFile == nil || l.file == nil {
		l.logger = slog.New(stdio)
		return
	}

	tmpl := ConsoleOptions{ShowTimestamp: true}
	if l.style != nil {
		l.style(&tmpl)
	}

	tmpl.Level = l.file
	tmpl.Writer = l.logFile
	tmpl.DisableColors = true
	tmpl.AddSource = true

	l.logger = slog.New(newTee(stdio, NewConsoleHandler(tmpl)))
}

// stdioHandler builds the console/stdout+stderr handler for the current format.
func (l *Logger) stdioHandler() slog.Handler {
	switch l.format {
	case FormatText:
		return slog.NewTextHandler(l.out, &slog.HandlerOptions{Level: l.stdio})
	case FormatJSON:
		return slog.NewJSONHandler(l.out, &slog.HandlerOptions{Level: l.stdio})
	default:
		tmpl := ConsoleOptions{ShowTimestamp: true}
		if l.style != nil {
			l.style(&tmpl)
		}

		tmpl.Level = l.stdio
		tmpl.Stdout = l.out
		tmpl.Stderr = l.err

		return NewConsoleHandler(tmpl)
	}
}

// SetLogFormat rebuilds the console/stdio stream in the given format (console,
// text or json). The file logger always stays plain text. It is meant to be set
// once at startup (eg. from a --log-format CLI flag); loggers already obtained
// via Named keep their previous format, so apply it before heavy logging. It is
// a no-op for a custom-handler logger (NewFromHandler) or an invalid format.
func (l *Logger) SetLogFormat(format Format) {
	if l.stdio == nil || !format.Valid() {
		return
	}

	l.format = format
	l.build()
}

// NewStdio returns a console-only logger (no log file):
//   - Info/Debug/Trace records are written to os.Stdout.
//   - Warn/Error/Fatal/Panic records are written to os.Stderr.
func NewStdio(level slog.Level) *Logger {
	levelVar := &slog.LevelVar{}
	levelVar.Set(level)

	out := newSwapWriter(os.Stdout)
	errOut := newSwapWriter(os.Stderr)

	console := NewConsoleHandler(ConsoleOptions{
		Level:         levelVar,
		Stdout:        out,
		Stderr:        errOut,
		ShowTimestamp: true,
	})

	return &Logger{
		logger: slog.New(console),
		stdio:  levelVar,
		out:    out,
		err:    errOut,
	}
}

// NewFromHandler wraps a user-provided slog.Handler as the sole logging backend.
// The console/file split and SetLevel knobs do not apply to such a logger.
func NewFromHandler(handler slog.Handler) *Logger {
	return &Logger{logger: slog.New(handler)}
}

// NewConsole returns a ready-to-use console *slog.Logger from the given options.
// It is the simplest way for a consumer to log with the team console style in its
// own code. Tag records with Named to get the aligned package column.
func NewConsole(opts ConsoleOptions) *slog.Logger {
	return slog.New(NewConsoleHandler(opts))
}

// Named tags a *slog.Logger with a package/domain and a more precise flow/stream,
// so the ConsoleHandler renders the aligned package column. It is sugar over
// logger.With(PackageKey, ...) / logger.With(StreamKey, ...).
func Named(logger *slog.Logger, pkg, stream string) *slog.Logger {
	return logger.With(
		slog.String(PackageKey, pkg),
		slog.String(StreamKey, stream),
	)
}

// Named returns a logger tagged with a package/domain and a more precise
// flow/stream, rendered by the console handler and recorded as attributes.
func (l *Logger) Named(pkg, stream string) *slog.Logger {
	return Named(l.logger, pkg, stream)
}

// Logger returns the underlying *slog.Logger.
func (l *Logger) Logger() *slog.Logger {
	return l.logger
}

// SetLevel adjusts the console and file logging levels at runtime. It is a no-op
// for loggers built from a custom handler (NewFromHandler).
func (l *Logger) SetLevel(level slog.Level) {
	if l.stdio != nil {
		l.stdio.Set(level)
	}

	// Keep the file at least as verbose: two sources are better than one when
	// debugging, and an in-memory filesystem makes this free.
	if l.file != nil {
		l.file.Set(level)
	}
}

// SetOutput redirects the console stdout and stderr streams at runtime (eg. onto
// a cobra command's output streams). A nil stream is left unchanged, and the call
// is a no-op for loggers built from a custom handler (NewFromHandler).
func (l *Logger) SetOutput(stdout, stderr io.Writer) {
	if l.out != nil {
		l.out.Set(stdout)
	}

	if l.err != nil {
		l.err.Set(stderr)
	}
}

// NewJSON returns a JSON-encoded logger writing to the given writer at the given
// level. The caller opens the destination (eg. a `<name>.json` file).
func NewJSON(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
}

// NewAudit returns a JSON-encoded audit logger writing to the given writer (eg. an
// opened audit.json file), at Debug level so every request is recorded.
func NewAudit(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// LevelFrom returns an slog.Level from an int, clamped to the [Trace, Panic] range.
func LevelFrom(level int) slog.Level {
	switch {
	case level < int(LevelTrace):
		return LevelTrace
	case level > int(LevelPanic):
		return LevelPanic
	default:
		return slog.Level(level)
	}
}

// FileName takes a filename without extension and adds
// the corresponding teamserver/teamclient logfile extension.
func FileName(name string, server bool) string {
	if server {
		return fmt.Sprintf("%s.%s", name, ServerLogFileExt)
	}

	return fmt.Sprintf("%s.%s", name, ClientLogFileExt)
}

// Trace logs a message at the custom Trace level.
func Trace(l *slog.Logger, msg string, args ...any) {
	l.Log(context.Background(), LevelTrace, msg, args...)
}

// Fatal logs a message at the custom Fatal level and then exits the program with
// status 1. It is reserved for unrecoverable failures (eg. the certificate
// infrastructure) where continuing would be unsafe.
func Fatal(l *slog.Logger, msg string, args ...any) {
	l.Log(context.Background(), LevelFatal, msg, args...)
	os.Exit(1)
}
