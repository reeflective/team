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
	"io"
	"log/slog"
)

// Format selects how console/stdio log records are rendered. All three formats
// are backed by the standard library (or this package) with no extra
// dependencies.
type Format string

const (
	// FormatConsole is the default: aligned, colored, human-readable output
	// rendered by ConsoleHandler.
	FormatConsole Format = "console"
	// FormatText is slog's TextHandler: uncolored logfmt "key=value" pairs,
	// friendly to grep/awk and log shippers.
	FormatText Format = "text"
	// FormatJSON is slog's JSONHandler: structured records for ingestion by log
	// pipelines (Loki, ELK, CloudWatch...).
	FormatJSON Format = "json"
)

// Formats returns all supported log formats, in a stable order. Useful to drive
// CLI flag validation and shell completion.
func Formats() []Format {
	return []Format{FormatConsole, FormatText, FormatJSON}
}

// Describe returns a one-line description of a format, for completion/help.
func (f Format) Describe() string {
	switch f {
	case FormatConsole:
		return "aligned, colored, human-readable console output"
	case FormatText:
		return "uncolored logfmt key=value pairs"
	case FormatJSON:
		return "structured JSON records for log pipelines"
	default:
		return ""
	}
}

// Valid reports whether f is a supported format.
func (f Format) Valid() bool {
	switch f {
	case FormatConsole, FormatText, FormatJSON:
		return true
	default:
		return false
	}
}

// String implements fmt.Stringer.
func (f Format) String() string {
	return string(f)
}

// NewFormatHandler builds a standalone slog.Handler rendering the given format to
// w at the given level. For FormatConsole it uses ConsoleHandler with the given
// style (nil for defaults); text/json use the stdlib handlers and ignore style.
//
// This is a convenience for consumers wiring their own logger; the team core
// builds its console/file handlers itself.
func NewFormatHandler(f Format, w io.Writer, level slog.Leveler, style *ConsoleOptions) slog.Handler {
	switch f {
	case FormatText:
		return slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	case FormatJSON:
		return slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	default:
		opts := ConsoleOptions{}
		if style != nil {
			opts = *style
		}

		opts.Level = level
		opts.Writer = w

		return NewConsoleHandler(opts)
	}
}
