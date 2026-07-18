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
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/carapace-sh/carapace/pkg/style"
)

// Custom slog levels. slog only defines Debug/Info/Warn/Error; the teamserver
// logging additionally uses a lower Trace level and higher Fatal/Panic levels
// (the latter two abort the program, and are only meant for unrecoverable
// failures such as the certificate infrastructure).
const (
	LevelTrace = slog.LevelDebug - 4 // -8
	LevelFatal = slog.LevelError + 4 // 12
	LevelPanic = slog.LevelError + 8 // 16
)

// Text effects.
const (
	sgrStart = "\x1b["
	sgrEnd   = "m"
)

const (
	// defaultTimeFormat is used when ShowTimestamp is on and no format is given.
	defaultTimeFormat = "15:04:05"

	// defaultPackageWidth is the column width the package/domain is padded to,
	// so that messages line up regardless of package name length.
	defaultPackageWidth = 10

	// defaultLevelWidth is the column width the level marker is padded to, so
	// that markers of different lengths (INFO vs ERROR) keep columns aligned.
	defaultLevelWidth = 5
)

// PackageKey is the attribute key identifying the name of the package
// (domain) specified by teamclients and teamservers named loggers.
const PackageKey = "teamserver_pkg"

// StreamKey is the attribute key identifying the more precise flow/stream
// within a named logger's package/domain.
const StreamKey = "stream"

// LevelStyle describes how a single log level is rendered on the console: a
// short label (eg. "[i]") and the color applied to it. The color is a
// carapace/style name (see github.com/carapace-sh/carapace/pkg/style).
//
// The whole point of this type is that level markers are trivially changeable:
// copy DefaultLevelStyles(), tweak the labels/colors you want, and set the
// result on ConsoleOptions.Levels.
type LevelStyle struct {
	Label string
	Color string
}

// DefaultLevelStyles returns the built-in level markers: the level word,
// colored per severity. They are padded to a uniform column (see
// defaultLevelWidth) so messages stay aligned and the output reads calmly.
//
// Restyling is meant to be trivial: copy this map, change the labels and/or
// colors you want (eg. to terse bracket markers like "[i]"/"[!]"), and set the
// result on ConsoleOptions.Levels — missing entries fall back to these defaults.
func DefaultLevelStyles() map[slog.Level]LevelStyle {
	return map[slog.Level]LevelStyle{
		LevelTrace:      {Label: "TRACE", Color: style.BrightBlack},
		slog.LevelDebug: {Label: "DEBUG", Color: style.Dim},
		slog.LevelInfo:  {Label: "INFO", Color: style.BrightBlue},
		slog.LevelWarn:  {Label: "WARN", Color: style.Yellow},
		slog.LevelError: {Label: "ERROR", Color: style.BrightRed},
		LevelFatal:      {Label: "FATAL", Color: style.Red},
		LevelPanic:      {Label: "PANIC", Color: style.BrightMagenta},
	}
}

// ConsoleHandler is a slog.Handler rendering aligned, colored log lines of the
// form:
//
//	[HH:MM:SS] <level-marker> <package> <message> [key=value ...]
//
// The timestamp is optional, the level marker and its color are configurable
// (see ConsoleOptions.Levels / LevelStyle), and the package column is padded so
// messages line up. Records are routed to stdout or stderr depending on their
// level (>= Warn goes to stderr), or to a single writer when one is set (used
// for file logging, in which case coloring is disabled and the caller is shown).
type ConsoleHandler struct {
	opts   ConsoleOptions
	levels map[slog.Level]LevelStyle
	mu     *sync.Mutex
	attrs  []slog.Attr
	group  string
}

// ConsoleOptions configures a ConsoleHandler.
type ConsoleOptions struct {
	// Level is the minimum level to log. Never nil for a working handler.
	Level slog.Leveler

	// Stdout/Stderr are the streams used for the level split. When Writer is
	// set, it takes precedence and both streams are ignored.
	Stdout io.Writer
	Stderr io.Writer

	// Writer, when set, receives all records (no stdout/stderr split). Used for
	// file logging.
	Writer io.Writer

	// Levels overrides the per-level markers/colors. Missing levels fall back to
	// DefaultLevelStyles(); a nil map uses the defaults entirely.
	Levels map[slog.Level]LevelStyle

	// PackageWidth is the column width the package/domain is padded to. Zero
	// uses defaultPackageWidth.
	PackageWidth int

	// LevelWidth is the column width the level marker is padded to. Zero uses
	// defaultLevelWidth.
	LevelWidth int

	// PackageColor, TimeColor and MessageColor are the carapace/style colors for
	// the package column, the timestamp and the message. Empty uses balanced
	// defaults (a muted/dimmed package, a faint timestamp, a bright message).
	// These make the non-level parts of a line as restyleable as the levels.
	PackageColor string
	TimeColor    string
	MessageColor string

	// DisableColors renders plain (uncolored) output. Automatically implied when
	// writing to a file.
	DisableColors bool

	// ShowTimestamp prepends a formatted timestamp to each line.
	ShowTimestamp bool

	// TimestampFormat is the layout used when ShowTimestamp is true. Empty uses
	// defaultTimeFormat.
	TimestampFormat string

	// AddSource renders the source [file:line] of the log call site.
	AddSource bool
}

// NewConsoleHandler returns a ConsoleHandler ready to use.
func NewConsoleHandler(opts ConsoleOptions) *ConsoleHandler {
	if opts.Level == nil {
		opts.Level = slog.LevelInfo
	}

	if opts.PackageWidth == 0 {
		opts.PackageWidth = defaultPackageWidth
	}

	if opts.LevelWidth == 0 {
		opts.LevelWidth = defaultLevelWidth
	}

	if opts.TimestampFormat == "" {
		opts.TimestampFormat = defaultTimeFormat
	}

	// Balanced defaults for the non-level columns: a muted (dimmed) cyan package
	// that stays legible, a faint grey timestamp, and a bright message.
	if opts.PackageColor == "" {
		opts.PackageColor = style.Of(style.Cyan, style.Dim)
	}

	if opts.TimeColor == "" {
		opts.TimeColor = style.BrightBlack
	}

	if opts.MessageColor == "" {
		opts.MessageColor = style.BrightWhite
	}

	// Resolve the effective level styles once (defaults + overrides).
	levels := DefaultLevelStyles()
	for level, ls := range opts.Levels {
		levels[level] = ls
	}

	return &ConsoleHandler{
		opts:   opts,
		levels: levels,
		mu:     &sync.Mutex{},
	}
}

// Enabled implements slog.Handler.
func (h *ConsoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

// WithAttrs implements slog.Handler.
func (h *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = make([]slog.Attr, 0, len(h.attrs)+len(attrs))
	clone.attrs = append(clone.attrs, h.attrs...)
	clone.attrs = append(clone.attrs, attrs...)

	return &clone
}

// WithGroup implements slog.Handler.
func (h *ConsoleHandler) WithGroup(name string) slog.Handler {
	clone := *h
	clone.group = name

	return &clone
}

// Handle implements slog.Handler: it formats and writes a single record.
func (h *ConsoleHandler) Handle(_ context.Context, rec slog.Record) error {
	var pkg string

	extra := make([]slog.Attr, 0, rec.NumAttrs())
	collect := func(a slog.Attr) {
		switch a.Key {
		case PackageKey:
			pkg = a.Value.String()
		case StreamKey:
			// Retained by structured handlers (JSON/audit) but, like the
			// original console formatter, not rendered on the console/file text.
		default:
			extra = append(extra, a)
		}
	}

	for _, a := range h.attrs {
		collect(a)
	}

	rec.Attrs(func(a slog.Attr) bool {
		collect(a)
		return true
	})

	line := h.format(rec, pkg, extra)

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := io.WriteString(h.writerFor(rec.Level), line)

	return err
}

// writerFor selects the output stream for a record level.
func (h *ConsoleHandler) writerFor(level slog.Level) io.Writer {
	if h.opts.Writer != nil {
		return h.opts.Writer
	}

	if level >= slog.LevelWarn {
		return h.opts.Stderr
	}

	return h.opts.Stdout
}

// levelStyle returns the marker/color for a level, falling back gracefully.
func (h *ConsoleHandler) levelStyle(level slog.Level) LevelStyle {
	if ls, ok := h.levels[level]; ok {
		return ls
	}

	return LevelStyle{Label: fmt.Sprintf("[%s]", level.String()), Color: style.Default}
}

// format assembles a single, aligned, optionally colored log line:
//
//	[time] <marker> <package> <message> [key=value ...]
func (h *ConsoleHandler) format(rec slog.Record, pkg string, extra []slog.Attr) string {
	colors := !h.opts.DisableColors && h.opts.Writer == nil

	var b strings.Builder

	if h.opts.ShowTimestamp {
		ts := rec.Time.Format(h.opts.TimestampFormat)
		b.WriteString(paint(colors, h.opts.TimeColor, ts))
		b.WriteByte(' ')
	}

	// Level marker, padded so markers of different lengths keep columns aligned.
	ls := h.levelStyle(rec.Level)
	b.WriteString(paint(colors, ls.Color, fmt.Sprintf("%-*s", h.opts.LevelWidth, ls.Label)))
	b.WriteByte(' ')

	// Package/domain, padded to a fixed column for message alignment.
	if pkg != "" {
		b.WriteString(paint(colors, h.opts.PackageColor, fmt.Sprintf("%-*s", h.opts.PackageWidth, pkg)))
		b.WriteByte(' ')
	}

	// Source [file:line], used by the file logger.
	if h.opts.AddSource && rec.PC != 0 {
		if src := sourceString(rec.PC); src != "" {
			b.WriteString(paint(colors, style.Dim, src))
			b.WriteByte(' ')
		}
	}

	b.WriteString(paint(colors, h.opts.MessageColor, rec.Message))

	for _, a := range extra {
		b.WriteByte(' ')
		b.WriteString(paint(colors, style.Dim, a.Key+"="))
		b.WriteString(a.Value.String())
	}

	b.WriteByte('\n')

	return b.String()
}

// sourceString renders a trimmed "file:line" for a program counter.
func sourceString(pc uintptr) string {
	fs := runtime.CallersFrames([]uintptr{pc})
	frame, _ := fs.Next()

	if frame.File == "" {
		return ""
	}

	file := frame.File
	if paths := strings.Split(file, "/mod/"); len(paths) > 1 && paths[1] != "" {
		file = filepath.Join(paths[1:]...)
	} else {
		file = filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file))
	}

	return fmt.Sprintf("[%s:%d]", file, frame.Line)
}

// paint wraps s in an SGR color sequence when colors are enabled.
func paint(enabled bool, c, s string) string {
	if !enabled || c == "" {
		return s
	}

	return color(c) + s + color(style.Default)
}

func color(c string) string {
	return sgrStart + style.SGR(c) + sgrEnd
}

// teeHandler dispatches each record to every sub-handler that enables its level.
// It is used to send events to both the console and the log file with
// independent levels.
type teeHandler struct {
	handlers []slog.Handler
}

// newTee returns a slog.Handler fanning out to all provided handlers.
func newTee(handlers ...slog.Handler) slog.Handler {
	return &teeHandler{handlers: handlers}
}

func (t *teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range t.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

func (t *teeHandler) Handle(ctx context.Context, rec slog.Record) error {
	for _, h := range t.handlers {
		if !h.Enabled(ctx, rec.Level) {
			continue
		}

		if err := h.Handle(ctx, rec); err != nil {
			return err
		}
	}

	return nil
}

func (t *teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}

	return &teeHandler{handlers: handlers}
}

func (t *teeHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		handlers[i] = h.WithGroup(name)
	}

	return &teeHandler{handlers: handlers}
}
