package commands_test

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
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/server"
	"github.com/reeflective/team/server/commands"
)

// These tests drive the real cobra command handlers in-process against an
// isolated, on-disk teamserver. They exist to catch the class of bug that unit
// tests on the library API miss entirely: panics and regressions inside the CLI
// handlers themselves (a nil certificate manager on `user`, a version-parsing
// panic on `systemd`, ...). Each command runs in a throwaway sandbox.

// newSandbox builds an isolated teamserver + self-teamclient whose commands can
// be executed in-process. Logging is discarded so no log file is held open
// (which would block t.TempDir() cleanup on Windows), and every path lives under
// a throwaway temp directory.
func newSandbox(t *testing.T) (*server.Server, *client.Client, string) {
	t.Helper()

	home := t.TempDir()
	discard := slog.NewTextHandler(io.Discard, nil)

	ts, err := server.New("test",
		server.WithHomeDirectory(home),
		server.WithLogger(discard),
	)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	tc := ts.Self(
		client.WithHomeDirectory(home),
		client.WithLogger(discard),
	)

	return ts, tc, home
}

// runCommand executes a single teamserver command against a freshly-generated
// command tree (cobra keeps per-tree flag state, so we rebuild each time) and
// captures its combined stdout/stderr. A panic in a handler fails the test
// naturally through the testing runtime.
func runCommand(t *testing.T, ts *server.Server, tc *client.Client, args ...string) (string, error) {
	t.Helper()

	root := commands.Generate(ts, tc)

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)

	err := root.Execute()

	return out.String(), err
}

// TestCommandUserLifecycle runs `user` then `delete` end to end. This is the
// path that previously panicked on a nil certificate manager.
func TestCommandUserLifecycle(t *testing.T) {
	ts, tc, home := newSandbox(t)

	out, err := runCommand(t, ts, tc, "user", "--name", "alice", "--host", "localhost", "--save", home)
	if err != nil {
		t.Fatalf("user create: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "alice") {
		t.Fatalf("expected the new identity details in output, got:\n%s", out)
	}

	configs, _ := filepath.Glob(filepath.Join(home, "*.teamclient.cfg"))
	if len(configs) == 0 {
		t.Fatalf("expected a *.teamclient.cfg to be written under %s", home)
	}

	out, err = runCommand(t, ts, tc, "delete", "alice")
	if err != nil {
		t.Fatalf("delete: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(strings.ToLower(out), "deleted") {
		t.Fatalf("expected a deletion confirmation, got:\n%s", out)
	}
}

// TestCommandSystemd is the regression guard for the version-parsing panic: the
// systemd config generator calls version.Semantic().
func TestCommandSystemd(t *testing.T) {
	ts, tc, _ := newSandbox(t)

	out, err := runCommand(t, ts, tc,
		"systemd", "--binpath", "/usr/bin/teamserver", "--user", "svc", "--port", "5432")
	if err != nil {
		t.Fatalf("systemd: %v\noutput:\n%s", err, out)
	}

	for _, want := range []string{"[Unit]", "[Service]", "ExecStart", "5432"} {
		if !strings.Contains(out, want) {
			t.Fatalf("systemd unit missing %q, got:\n%s", want, out)
		}
	}
}

// TestCommandStatus renders the teamserver status view.
func TestCommandStatus(t *testing.T) {
	ts, tc, _ := newSandbox(t)

	out, err := runCommand(t, ts, tc, "status")
	if err != nil {
		t.Fatalf("status: %v\noutput:\n%s", err, out)
	}
	if !strings.Contains(out, "General") {
		t.Fatalf("status output looks wrong:\n%s", out)
	}
}

// TestCommandExportCA exports the user certificate authority to disk, which
// forces lazy certificate-infrastructure initialization through the CLI.
func TestCommandExportCA(t *testing.T) {
	ts, tc, home := newSandbox(t)

	dest := filepath.Join(home, "ca")
	if err := os.MkdirAll(dest, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	out, err := runCommand(t, ts, tc, "export", dest)
	if err != nil {
		t.Fatalf("export: %v\noutput:\n%s", err, out)
	}

	files, _ := filepath.Glob(filepath.Join(dest, "*"))
	if len(files) == 0 {
		t.Fatalf("expected an exported CA file under %s", dest)
	}
}

// TestCommandGuide renders the built-in usage guide.
func TestCommandGuide(t *testing.T) {
	ts, tc, _ := newSandbox(t)

	out, err := runCommand(t, ts, tc, "guide")
	if err != nil {
		t.Fatalf("guide: %v", err)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("guide printed nothing")
	}
}
