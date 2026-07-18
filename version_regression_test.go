package team_test

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
	"runtime"
	"testing"

	"github.com/reeflective/team/client"
	grpcserver "github.com/reeflective/team/example/transports/grpc/server"
	"github.com/reeflective/team/server"
)

// TestVersionClientNoRecursion reproduces the exact setup that previously caused
// infinite recursion / stack overflow: a teamclient whose team.Client backend is
// an RPC transport that embeds the very same *client.Client core (so the promoted
// VersionClient() pointed back at the core). VersionClient() must now resolve
// locally and never traverse the backend.
func TestVersionClientNoRecursion(t *testing.T) {
	// A gRPC listener is only needed to build the in-memory dialer/backend.
	gtc := grpcserver.NewClientFrom(grpcserver.NewListener())

	// No positional backend: the dialer implements team.Client and is picked up
	// automatically as the backend (finding 2).
	core, err := client.New("versiontest", client.WithDialer(gtc))
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	// Simulate what Connect() does: inject the core into the dialer, so the
	// transport's embedded *client.Client points back at the core. This is the
	// arrangement that used to recurse.
	if err := gtc.Init(core); err != nil {
		t.Fatalf("dialer Init: %v", err)
	}

	// Must return the local binary version without recursing.
	ver, err := core.VersionClient()
	if err != nil {
		t.Fatalf("VersionClient returned error: %v", err)
	}

	if ver.OS != runtime.GOOS || ver.Arch != runtime.GOARCH {
		t.Fatalf("VersionClient should report the local runtime, got OS=%q Arch=%q", ver.OS, ver.Arch)
	}
}

// TestSelfClientVersionFlow exercises the full in-memory self-client flow (server
// serving itself over the gRPC bufconn) and checks that the three team.Client
// surface calls answer without panicking: client version (local), server version
// (through the in-memory backend), and users.
func TestSelfClientVersionFlow(t *testing.T) {
	gts := grpcserver.NewListener()

	ts, err := server.New("selftest", server.WithHandler(gts), server.WithInMemory())
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	gtc := grpcserver.NewClientFrom(gts)

	// Self() registers the server itself as the team.Client backend (finding 2),
	// while the gRPC dialer drives the in-memory connection.
	self := ts.Self(client.WithDialer(gtc))

	if err := ts.Serve(self); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	defer self.Disconnect()

	if _, err := self.VersionClient(); err != nil {
		t.Fatalf("VersionClient: %v", err)
	}

	if _, err := self.VersionServer(); err != nil {
		t.Fatalf("VersionServer: %v", err)
	}

	if _, err := self.Users(); err != nil {
		t.Fatalf("Users: %v", err)
	}
}
