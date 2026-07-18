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

func TestFormatValid(t *testing.T) {
	for _, f := range Formats() {
		if !f.Valid() {
			t.Fatalf("Formats() returned an invalid format: %q", f)
		}
	}

	if Format("yaml").Valid() {
		t.Fatal("unexpected format should be invalid")
	}

	if len(Formats()) != 3 {
		t.Fatalf("expected 3 formats, got %d", len(Formats()))
	}
}

func TestSetLogFormat(t *testing.T) {
	logger := NewStdio(slog.LevelInfo)

	var buf bytes.Buffer
	logger.SetOutput(&buf, &buf)

	// text -> logfmt key=value
	logger.SetLogFormat(FormatText)
	logger.Named("server", "main").Info("hello")

	if !strings.Contains(buf.String(), "level=INFO") || !strings.Contains(buf.String(), "msg=hello") {
		t.Fatalf("text format should render logfmt, got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "teamserver_pkg=server") {
		t.Fatalf("text format should render the package attr, got: %q", buf.String())
	}
	buf.Reset()

	// json -> structured
	logger.SetLogFormat(FormatJSON)
	logger.Named("server", "main").Info("hi")

	if !strings.Contains(buf.String(), `"msg":"hi"`) || !strings.Contains(buf.String(), `"level":"INFO"`) {
		t.Fatalf("json format should render structured records, got: %q", buf.String())
	}
	buf.Reset()

	// back to console -> our level word + package column
	logger.SetLogFormat(FormatConsole)
	logger.Named("server", "main").Info("back")

	if !strings.Contains(buf.String(), "INFO") || !strings.Contains(buf.String(), "server") {
		t.Fatalf("console format should render the team console, got: %q", buf.String())
	}
	if strings.Contains(buf.String(), "msg=back") {
		t.Fatalf("console format should not be logfmt, got: %q", buf.String())
	}
}

func TestSetLogFormatInvalidIsNoop(t *testing.T) {
	logger := NewStdio(slog.LevelInfo)

	var buf bytes.Buffer
	logger.SetOutput(&buf, &buf)
	logger.SetLogFormat(FormatJSON)
	logger.SetLogFormat(Format("bogus")) // ignored, stays json

	logger.Named("server", "main").Info("x")
	if !strings.Contains(buf.String(), `"msg":"x"`) {
		t.Fatalf("invalid format should be a no-op keeping json, got: %q", buf.String())
	}
}

func TestNewFormatHandler(t *testing.T) {
	var buf bytes.Buffer

	json := slog.New(NewFormatHandler(FormatJSON, &buf, slog.LevelInfo, nil))
	json.Info("j")
	if !strings.Contains(buf.String(), `"msg":"j"`) {
		t.Fatalf("NewFormatHandler json wrong: %q", buf.String())
	}
	buf.Reset()

	console := slog.New(NewFormatHandler(FormatConsole, &buf, slog.LevelInfo, nil))
	Named(console, "app", "").Info("c")
	if !strings.Contains(buf.String(), "INFO") || !strings.Contains(buf.String(), "app") {
		t.Fatalf("NewFormatHandler console wrong: %q", buf.String())
	}
}
