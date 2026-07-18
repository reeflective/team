package commands

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
	"io/fs"
	"log/slog"
	"testing"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace/pkg/style"

	"github.com/reeflective/team/client"
)

// fakeDirEntry is a minimal fs.DirEntry for exercising isConfigDir without
// touching the filesystem.
type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f fakeDirEntry) Name() string { return f.name }
func (f fakeDirEntry) IsDir() bool  { return f.isDir }
func (f fakeDirEntry) Type() fs.FileMode {
	if f.isDir {
		return fs.ModeDir
	}
	return 0
}
func (f fakeDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

// TestIsConfigDir guards the config-directory filter used by the import
// completers. The regression it protects against: the condition was inverted so
// that, with noSelf=true, no directory ever matched and cross-application config
// completion silently returned nothing.
func TestIsConfigDir(t *testing.T) {
	cli, err := client.New("myapp",
		client.WithInMemory(),
		client.WithLogger(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	cases := []struct {
		name   string
		entry  fakeDirEntry
		noSelf bool
		want   bool
	}{
		{"other app is included (noSelf)", fakeDirEntry{".otherapp", true}, true, true},
		{"own app is excluded (noSelf)", fakeDirEntry{".myapp", true}, true, false},
		{"own app is included (self allowed)", fakeDirEntry{".myapp", true}, false, true},
		{"non-hidden dir is ignored", fakeDirEntry{"otherapp", true}, true, false},
		{"hidden non-dir is ignored", fakeDirEntry{".otherapp", false}, true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isConfigDir(cli, tc.entry, tc.noSelf); got != tc.want {
				t.Fatalf("isConfigDir(%q, noSelf=%v) = %v, want %v", tc.entry.name, tc.noSelf, got, tc.want)
			}
		})
	}
}

// TestGetConfigStyle checks that config/CA files are highlighted while every
// other path keeps carapace's default per-type styling (so directories stay
// blue, etc.) rather than being stripped of color.
func TestGetConfigStyle(t *testing.T) {
	styleFn := GetConfigStyle(".teamclient.cfg")
	ctx := carapace.Context{}

	if got := styleFn("alice.teamclient.cfg", ctx); got != style.Red {
		t.Fatalf("config file style = %q, want %q", got, style.Red)
	}

	// Non-config paths must defer to carapace's default path styling, not "" and
	// not the value itself.
	for _, path := range []string{"notes.txt", "somedir/", "/usr/bin/env"} {
		if got, want := styleFn(path, ctx), style.ForPath(path, ctx); got != want {
			t.Fatalf("non-config %q style = %q, want ForPath %q", path, got, want)
		}
	}
}
