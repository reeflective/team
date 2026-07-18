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
	"log/slog"
	"strings"
	"testing"
)

func TestLevelFrom(t *testing.T) {
	cases := map[int]slog.Level{
		int(LevelTrace) - 100: LevelTrace, // clamped up to Trace
		int(LevelTrace):       LevelTrace,
		int(slog.LevelInfo):   slog.LevelInfo,
		int(slog.LevelError):  slog.LevelError,
		int(LevelPanic) + 100: LevelPanic, // clamped down to Panic
	}

	for in, want := range cases {
		if got := LevelFrom(in); got != want {
			t.Fatalf("LevelFrom(%d) = %v, want %v", in, got, want)
		}
	}
}

func TestLoggerSetLevelAndOutput(t *testing.T) {
	logger := NewStdio(slog.LevelWarn)

	var out, errOut bytes.Buffer
	logger.SetOutput(&out, &errOut)

	// Below threshold: dropped.
	logger.Logger().Info("hidden")
	if out.Len() != 0 {
		t.Fatalf("info should be filtered at warn level, got: %q", out.String())
	}

	// Lower the level at runtime; info now passes to stdout.
	logger.SetLevel(slog.LevelInfo)
	logger.Logger().Info("now-visible")
	if !strings.Contains(out.String(), "now-visible") {
		t.Fatalf("info should appear after SetLevel, got: %q", out.String())
	}

	// Warnings route to stderr.
	logger.Logger().Warn("warned")
	if !strings.Contains(errOut.String(), "warned") {
		t.Fatalf("warn should route to stderr, got: %q", errOut.String())
	}
}

func TestLoggerNamedRedactsStream(t *testing.T) {
	logger := NewStdio(slog.LevelDebug)

	var out bytes.Buffer
	logger.SetOutput(&out, &out)

	logger.Named("server", "listeners").Info("up")
	if !strings.Contains(out.String(), "server") {
		t.Fatalf("expected package name, got: %q", out.String())
	}
	if strings.Contains(out.String(), "listeners") {
		t.Fatalf("stream should not render on console, got: %q", out.String())
	}
}

func TestNewTeeToFileAndConsole(t *testing.T) {
	// The "log file" is any io.Writer — a buffer is enough to assert content.
	var file bytes.Buffer

	// Console at Warn, file at Debug: a debug record must reach the file only.
	logger := New(&file, slog.LevelWarn, slog.LevelDebug, nil)

	var console bytes.Buffer
	logger.SetOutput(&console, &console)

	logger.Named("server", "main").Debug("file-only-debug")
	logger.Named("server", "main").Error("both-error")

	// File captured both (its level is Debug).
	fileOut := file.String()
	if !strings.Contains(fileOut, "file-only-debug") || !strings.Contains(fileOut, "both-error") {
		t.Fatalf("file should contain both records, got: %q", fileOut)
	}

	// Console (Warn) dropped the debug but kept the error.
	if strings.Contains(console.String(), "file-only-debug") {
		t.Fatalf("console should not contain the debug record, got: %q", console.String())
	}
	if !strings.Contains(console.String(), "both-error") {
		t.Fatalf("console should contain the error record, got: %q", console.String())
	}

	// The file logger renders the source [file:line] (AddSource).
	if !strings.Contains(fileOut, "log_test.go:") {
		t.Fatalf("file logger should render source caller, got: %q", fileOut)
	}
}

func TestNewFromHandler(t *testing.T) {
	var buf bytes.Buffer
	h := NewConsoleHandler(ConsoleOptions{Level: slog.LevelInfo, Stdout: &buf, Stderr: &buf, DisableColors: true})

	logger := NewFromHandler(h)
	// SetLevel/SetOutput are no-ops for a custom handler and must not panic.
	logger.SetLevel(slog.LevelDebug)
	logger.SetOutput(&buf, &buf)

	logger.Logger().Info("custom-backend")
	if !strings.Contains(buf.String(), "custom-backend") {
		t.Fatalf("expected message through custom handler, got: %q", buf.String())
	}
}
