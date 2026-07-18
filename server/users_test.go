package server

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
	"crypto/tls"
	"errors"
	"testing"
)

// newTestServer returns a fully-initialized in-memory teamserver. Calling init()
// bootstraps the database and certificate infrastructure without needing a
// transport handler, which is all the user-management primitives require.
func newTestServer(t *testing.T) *Server {
	t.Helper()

	ts, err := New("usertest", WithInMemory())
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	if err := ts.init(); err != nil {
		t.Fatalf("server.init: %v", err)
	}

	return ts
}

// TestUserCreateWithoutServe is a regression guard for the nil-certificate
// panic: creating a user through the API (as the `user` CLI command does) must
// work without a prior Serve()/init() call. Previously UserCreate only ran
// initDatabase(), while ts.certs was built solely on the serve path, so this
// dereferenced a nil *certs.Manager.
func TestUserCreateWithoutServe(t *testing.T) {
	ts, err := New("nocerts", WithInMemory())
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}

	// Deliberately NOT calling ts.init() / Serve() here.
	cfg, err := ts.UserCreate("alice", "localhost", 31337)
	if err != nil {
		t.Fatalf("UserCreate without serve: %v", err)
	}
	if cfg == nil || cfg.Token == "" {
		t.Fatal("expected a valid client config with a non-empty token")
	}

	// The freshly-minted identity must authenticate.
	if u, err := ts.Authenticate(cfg.Token); err != nil || u == nil || u.Name != "alice" {
		t.Fatalf("authenticate minted identity: user=%v err=%v", u, err)
	}
}

// TestUserCreateValidation pins the input validation on UserCreate: user names
// are restricted to alphanumerics (plus - and _), and neither the name nor the
// host may be empty. All rejections surface as ErrUserConfig.
func TestUserCreateValidation(t *testing.T) {
	ts := newTestServer(t)

	cases := []struct {
		name  string
		user  string
		lhost string
	}{
		{"empty name", "", "localhost"},
		{"empty host", "alice", ""},
		{"space in name", "alice bob", "localhost"},
		{"slash in name", "alice/bob", "localhost"},
		{"dot in name", "alice.bob", "localhost"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ts.UserCreate(tc.user, tc.lhost, 31337)
			if err == nil {
				t.Fatalf("UserCreate(%q, %q) should have failed", tc.user, tc.lhost)
			}
			if !errors.Is(err, ErrUserConfig) {
				t.Fatalf("expected ErrUserConfig, got %v", err)
			}
			if cfg != nil {
				t.Fatal("expected nil config on validation failure")
			}
		})
	}
}

// TestUserCreateTokenUniqueness ensures two users minted back-to-back receive
// distinct API tokens and distinct client certificates, and that each token
// authenticates back to the right identity.
func TestUserCreateTokenUniqueness(t *testing.T) {
	ts := newTestServer(t)

	alice, err := ts.UserCreate("alice", "localhost", 31337)
	if err != nil {
		t.Fatalf("UserCreate(alice): %v", err)
	}
	bob, err := ts.UserCreate("bob", "localhost", 31337)
	if err != nil {
		t.Fatalf("UserCreate(bob): %v", err)
	}

	if alice.Token == "" || bob.Token == "" {
		t.Fatal("tokens must be non-empty")
	}
	if alice.Token == bob.Token {
		t.Fatal("two users must not share the same API token")
	}
	if alice.Certificate == bob.Certificate {
		t.Fatal("two users must not share the same client certificate")
	}

	if u, err := ts.Authenticate(alice.Token); err != nil || u == nil || u.Name != "alice" {
		t.Fatalf("alice token must authenticate as alice, got user=%v err=%v", u, err)
	}
	if u, err := ts.Authenticate(bob.Token); err != nil || u == nil || u.Name != "bob" {
		t.Fatalf("bob token must authenticate as bob, got user=%v err=%v", u, err)
	}
}

// TestUserDeleteRevokesAuth is the security-critical guarantee documented on
// UserDelete: once a user is deleted, its token no longer authenticates (even
// though it was previously cached) and its client certificate is gone.
func TestUserDeleteRevokesAuth(t *testing.T) {
	ts := newTestServer(t)

	cfg, err := ts.UserCreate("mallory", "localhost", 31337)
	if err != nil {
		t.Fatalf("UserCreate: %v", err)
	}

	// Authenticate once so the token is now in the in-memory cache; deletion
	// must invalidate the cache too, not just the database row.
	if _, err := ts.Authenticate(cfg.Token); err != nil {
		t.Fatalf("Authenticate before delete: %v", err)
	}

	// The client certificate must exist before deletion.
	if _, _, err := ts.certs.UserClientGetCertificate("mallory"); err != nil {
		t.Fatalf("client certificate should exist before delete: %v", err)
	}

	if err := ts.UserDelete("mallory"); err != nil {
		t.Fatalf("UserDelete: %v", err)
	}

	// Token must no longer authenticate.
	user, err := ts.Authenticate(cfg.Token)
	if err == nil || user != nil {
		t.Fatalf("deleted user's token must be rejected, got user=%v err=%v", user, err)
	}
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected ErrUnauthenticated after delete, got %v", err)
	}

	// The client certificate must be gone.
	if _, _, err := ts.certs.UserClientGetCertificate("mallory"); err == nil {
		t.Fatal("client certificate should have been removed on delete")
	}
}

// TestUsersTLSConfig verifies the server-side mutual-TLS configuration is locked
// down: it requires and verifies client certificates, pins TLS 1.3, and carries
// exactly one server certificate plus a client CA pool.
func TestUsersTLSConfig(t *testing.T) {
	ts := newTestServer(t)

	tlsConfig, err := ts.UsersTLSConfig()
	if err != nil {
		t.Fatalf("UsersTLSConfig: %v", err)
	}

	if tlsConfig.ClientAuth != tls.RequireAndVerifyClientCert {
		t.Fatalf("expected RequireAndVerifyClientCert, got %v", tlsConfig.ClientAuth)
	}
	if tlsConfig.MinVersion != tls.VersionTLS13 {
		t.Fatalf("expected MinVersion TLS 1.3, got %x", tlsConfig.MinVersion)
	}
	if len(tlsConfig.Certificates) != 1 {
		t.Fatalf("expected exactly 1 server certificate, got %d", len(tlsConfig.Certificates))
	}
	if tlsConfig.ClientCAs == nil || tlsConfig.RootCAs == nil {
		t.Fatal("expected both a client CA pool and a root CA pool to be set")
	}
}

// TestAuthenticateRejectsGarbage ensures a well-formed-but-unknown token and an
// empty token are both rejected without leaking an identity.
func TestAuthenticateRejectsGarbage(t *testing.T) {
	ts := newTestServer(t)

	for _, tok := range []string{"", "deadbeef", "not-a-real-token"} {
		user, err := ts.Authenticate(tok)
		if err == nil || user != nil {
			t.Fatalf("token %q must be rejected, got user=%v err=%v", tok, user, err)
		}
	}
}
