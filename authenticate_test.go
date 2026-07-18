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
	"testing"

	"github.com/reeflective/team/client"
	grpcserver "github.com/reeflective/team/example/transports/grpc/server"
	"github.com/reeflective/team/server"
)

// TestAuthenticatePrimitive pins down the authentication-only contract of the
// teamserver (direction C): UserCreate issues an identity + its credentials, and
// Authenticate is a pure verification primitive that maps a raw token back to the
// registered identity — carrying no authorization data. Authorization is left
// entirely to the embedding application.
func TestAuthenticatePrimitive(t *testing.T) {
	gts := grpcserver.NewListener()

	ts, err := server.New("authtest", server.WithHandler(gts), server.WithInMemory())
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	// Serving the in-memory self triggers DB + certificate initialization, which
	// UserCreate needs to mint the user's client certificate.
	gtc := grpcserver.NewClientFrom(gts)
	self := ts.Self(client.WithDialer(gtc))

	if err := ts.Serve(self); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	defer self.Disconnect()

	// The teamserver issues the identity and its credentials (token + certs).
	cfg, err := ts.UserCreate("alice", "localhost", 31337)
	if err != nil {
		t.Fatalf("UserCreate: %v", err)
	}
	if cfg.Token == "" {
		t.Fatal("UserCreate must return a non-empty API token")
	}

	// A valid token resolves to the registered identity...
	user, err := ts.Authenticate(cfg.Token)
	if err != nil {
		t.Fatalf("Authenticate(valid token): %v", err)
	}
	if user == nil || user.Name != "alice" {
		t.Fatalf("expected authenticated identity %q, got %+v", "alice", user)
	}

	// ...and the identity is authentication-only: team.User exposes no
	// permission/role surface at all (this is enforced at compile time — the
	// field no longer exists — so there is nothing to assert here beyond it).

	// A bogus token must be rejected, leaking no identity.
	if bad, err := ts.Authenticate("not-a-real-token"); err == nil || bad != nil {
		t.Fatalf("invalid token must be rejected, got user=%+v err=%v", bad, err)
	}
}
