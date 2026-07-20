package boot

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
	"testing"

	"github.com/reeflective/team/client"
)

// isolateHome points the app's root directory at an empty temp dir so config
// discovery finds nothing (deterministic ModeServer) and no real user config
// leaks into the test.
func isolateHome(t *testing.T, app string) {
	t.Helper()
	t.Setenv("TESTAPP_ROOT_DIR", t.TempDir())
}

func TestResolveForceServerWins(t *testing.T) {
	isolateHome(t, "testapp")

	// Even with a pinned client config, ForceServer must select server mode.
	b := Boot{
		App:         "testapp",
		ForceServer: true,
		Config:      &client.Config{User: "alice", Host: "10.0.0.1", Port: 31337},
		Client:      func(*client.Config) error { return nil },
		Server:      func() error { return nil },
	}

	if mode, cfg := b.Resolve(); mode != ModeServer || cfg != nil {
		t.Fatalf("ForceServer: got mode=%v cfg=%v, want ModeServer/nil", mode, cfg)
	}
}

func TestResolveExplicitConfigSelectsClient(t *testing.T) {
	isolateHome(t, "testapp")

	pinned := &client.Config{User: "bob", Host: "10.0.0.2", Port: 32333}
	b := Boot{
		App:    "testapp",
		Config: pinned,
		Client: func(*client.Config) error { return nil },
		Server: func() error { return nil },
	}

	mode, cfg := b.Resolve()
	if mode != ModeClient {
		t.Fatalf("explicit config: got mode=%v, want ModeClient", mode)
	}
	if cfg != pinned {
		t.Fatalf("explicit config: got cfg=%v, want the pinned config", cfg)
	}
}

func TestResolveNoConfigDefaultsToServer(t *testing.T) {
	isolateHome(t, "testapp")

	b := Boot{
		App:    "testapp",
		Client: func(*client.Config) error { return nil },
		Server: func() error { return nil },
	}

	if mode, cfg := b.Resolve(); mode != ModeServer || cfg != nil {
		t.Fatalf("no config: got mode=%v cfg=%v, want ModeServer/nil", mode, cfg)
	}
}

func TestRunDispatchesToResolvedCallback(t *testing.T) {
	isolateHome(t, "testapp")

	// Explicit config -> client callback runs, server callback must not.
	var ranClient, ranServer bool
	err := Run(Boot{
		App:    "testapp",
		Config: &client.Config{User: "carol", Host: "h", Port: 1},
		Client: func(*client.Config) error { ranClient = true; return nil },
		Server: func() error { ranServer = true; return nil },
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !ranClient || ranServer {
		t.Fatalf("client mode: ranClient=%v ranServer=%v, want true/false", ranClient, ranServer)
	}

	// No config, isolated home -> server callback runs.
	ranClient, ranServer = false, false
	err = Run(Boot{
		App:    "testapp",
		Client: func(*client.Config) error { ranClient = true; return nil },
		Server: func() error { ranServer = true; return nil },
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if ranClient || !ranServer {
		t.Fatalf("server mode: ranClient=%v ranServer=%v, want false/true", ranClient, ranServer)
	}
}

func TestRunValidatesRequiredFields(t *testing.T) {
	cases := map[string]Boot{
		"missing app":    {Client: func(*client.Config) error { return nil }, Server: func() error { return nil }},
		"missing client": {App: "testapp", Server: func() error { return nil }},
		"missing server": {App: "testapp", Client: func(*client.Config) error { return nil }},
	}

	for name, b := range cases {
		if err := Run(b); err == nil {
			t.Fatalf("%s: expected an error, got nil", name)
		}
	}
}
