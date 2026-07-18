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
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// newTestLogger returns an slog.Logger writing uncolored output to buf, at the
// given level, so assertions can match plain text.
func newTestLogger(buf *bytes.Buffer, level slog.Leveler, opts ConsoleOptions) *slog.Logger {
	opts.Level = level
	opts.Stdout = buf
	opts.Stderr = buf
	opts.DisableColors = true

	return slog.New(NewConsoleHandler(opts))
}

func TestConsoleHandlerFormat(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf, slog.LevelDebug, ConsoleOptions{}).
		With(slog.String(PackageKey, "server"), slog.String(StreamKey, "main"))

	log.Info("serving", "addr", ":31337")
	out := buf.String()

	// Default (uncolored) line: "<LEVEL padded> <package padded> <message> attr=val".
	if !strings.Contains(out, "INFO") {
		t.Fatalf("expected level word INFO, got: %q", out)
	}
	if !strings.Contains(out, "server") {
		t.Fatalf("expected package, got: %q", out)
	}
	if !strings.Contains(out, "serving") {
		t.Fatalf("expected message, got: %q", out)
	}
	if !strings.Contains(out, "addr=:31337") {
		t.Fatalf("expected extra attr rendered, got: %q", out)
	}
	// The stream field must NOT be rendered on the console line.
	if strings.Contains(out, "main") {
		t.Fatalf("stream field should not be rendered on console, got: %q", out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("line should end with newline, got: %q", out)
	}
}

func TestConsoleHandlerAlignment(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf, slog.LevelDebug, ConsoleOptions{})

	log.With(slog.String(PackageKey, "db")).Info("a")
	log.With(slog.String(PackageKey, "certs")).Error("b")
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), buf.String())
	}

	// The message column must start at the same offset regardless of the
	// (differing) level and package widths, because both are padded.
	idxA := strings.Index(lines[0], "a")
	idxB := strings.Index(lines[1], "b")
	if idxA != idxB {
		t.Fatalf("messages not aligned: 'a'@%d vs 'b'@%d (%q | %q)", idxA, idxB, lines[0], lines[1])
	}
}

func TestConsoleHandlerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf, slog.LevelWarn, ConsoleOptions{})

	log.Info("hidden")
	log.Warn("shown")

	out := buf.String()
	if strings.Contains(out, "hidden") {
		t.Fatalf("info should be filtered below warn, got: %q", out)
	}
	if !strings.Contains(out, "shown") {
		t.Fatalf("warn should be logged, got: %q", out)
	}
}

func TestConsoleHandlerStdoutStderrSplit(t *testing.T) {
	var out, errBuf bytes.Buffer
	h := NewConsoleHandler(ConsoleOptions{
		Level:         LevelTrace,
		Stdout:        &out,
		Stderr:        &errBuf,
		DisableColors: true,
	})
	log := slog.New(h)

	log.Info("to-stdout")
	log.Error("to-stderr")

	if !strings.Contains(out.String(), "to-stdout") || strings.Contains(out.String(), "to-stderr") {
		t.Fatalf("stdout stream wrong: %q", out.String())
	}
	if !strings.Contains(errBuf.String(), "to-stderr") || strings.Contains(errBuf.String(), "to-stdout") {
		t.Fatalf("stderr stream wrong: %q", errBuf.String())
	}
}

func TestConsoleHandlerCustomLevelStyle(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf, slog.LevelDebug, ConsoleOptions{
		Levels: map[slog.Level]LevelStyle{
			slog.LevelInfo: {Label: "[i]", Color: "cyan"},
		},
	})

	log.Info("hello")
	log.Warn("world") // not overridden -> falls back to default "WARN"

	out := buf.String()
	if !strings.Contains(out, "[i]") {
		t.Fatalf("expected overridden info marker [i], got: %q", out)
	}
	if !strings.Contains(out, "WARN") {
		t.Fatalf("expected default WARN fallback, got: %q", out)
	}
}

func TestConsoleHandlerTimestamp(t *testing.T) {
	var buf bytes.Buffer
	log := newTestLogger(&buf, slog.LevelInfo, ConsoleOptions{
		ShowTimestamp:   true,
		TimestampFormat: "2006",
	})

	log.Info("msg")
	if !strings.Contains(buf.String(), "20") { // year prefix
		t.Fatalf("expected timestamp, got: %q", buf.String())
	}
}

// TestVisual prints colored sample output for manual inspection with:
//
//	go test ./internal/log -run TestVisual -v
func TestVisual(t *testing.T) {
	if os.Getenv("LOG_VISUAL") == "" {
		t.Skip("set LOG_VISUAL=1 to print colored samples")
	}

	lv := &slog.LevelVar{}
	lv.Set(LevelTrace)
	log := slog.New(NewConsoleHandler(ConsoleOptions{Level: lv, Stdout: os.Stdout, Stderr: os.Stdout, ShowTimestamp: true}))

	log.With(slog.String(PackageKey, "server")).Log(context.Background(), LevelTrace, "trace sample")
	log.With(slog.String(PackageKey, "config")).Debug("loaded ~/.app/teamserver.cfg")
	log.With(slog.String(PackageKey, "server")).Info("Serving teamserver on :31337")
	log.With(slog.String(PackageKey, "certs")).Warn("regenerating expired users CA")
	log.With(slog.String(PackageKey, "database")).Error("dial tcp :5432: connection refused", "attempt", 3)
}
