package db

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
	"errors"
	"net/url"
	"strings"
	"testing"
)

// TestDSNSqlite pins the SQLite DSN format: a file: URI with the database path
// and encoded params. This is the default/in-memory backend and the one path
// the integration tests already exercise, kept here as an explicit contract.
func TestDSNSqlite(t *testing.T) {
	cfg := &Config{
		Dialect:  Sqlite,
		Database: SQLiteInMemoryHost,
		Params:   map[string]string{"cache": "shared"},
	}

	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("DSN(sqlite): unexpected error %v", err)
	}

	if !strings.HasPrefix(dsn, "file::memory:?") {
		t.Fatalf("sqlite DSN must be a file: URI for the in-memory host, got %q", dsn)
	}
	if !strings.Contains(dsn, "cache=shared") {
		t.Fatalf("sqlite DSN must carry encoded params, got %q", dsn)
	}
}

// TestDSNSqliteEncrypted verifies that supplying an encryption key routes an
// on-disk SQLite database through the adiantum VFS, with the key carried as a
// (URL-encoded) textkey parameter.
func TestDSNSqliteEncrypted(t *testing.T) {
	cfg := &Config{
		Dialect:       Sqlite,
		Database:      "/var/lib/team/app.db",
		EncryptionKey: "s3cr3t key/with=chars",
	}

	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("DSN(encrypted sqlite): %v", err)
	}

	if !strings.Contains(dsn, "vfs=adiantum") {
		t.Fatalf("encrypted DSN must select the adiantum VFS, got %q", dsn)
	}
	if !strings.Contains(dsn, "textkey="+url.QueryEscape("s3cr3t key/with=chars")) {
		t.Fatalf("encrypted DSN must carry the URL-encoded textkey, got %q", dsn)
	}
	// The raw key with its unescaped special characters must not appear.
	if strings.Contains(dsn, "s3cr3t key/with=chars") {
		t.Fatalf("encrypted DSN leaked an unescaped key, got %q", dsn)
	}
}

// TestDSNSqliteInMemoryNotEncrypted ensures the encryption key is ignored for
// in-memory databases: there is nothing on disk to protect, and selecting the
// adiantum VFS there would only add overhead.
func TestDSNSqliteInMemoryNotEncrypted(t *testing.T) {
	cfg := &Config{
		Dialect:       Sqlite,
		Database:      SQLiteInMemoryHost,
		EncryptionKey: "ignored-for-memory",
	}

	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("DSN(in-memory): %v", err)
	}
	if strings.Contains(dsn, "adiantum") || strings.Contains(dsn, "textkey") {
		t.Fatalf("in-memory DSN must not be encrypted, got %q", dsn)
	}
}

// TestDSNSqlitePlaintextByDefault pins the opt-in contract: with no key, the
// DSN is the plain file: URI with no VFS selected.
func TestDSNSqlitePlaintextByDefault(t *testing.T) {
	cfg := &Config{Dialect: Sqlite, Database: "/var/lib/team/app.db"}

	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("DSN(plaintext): %v", err)
	}
	if strings.Contains(dsn, "adiantum") || strings.Contains(dsn, "textkey") {
		t.Fatalf("default DSN must be plaintext (opt-in encryption), got %q", dsn)
	}
}

// TestDSNMySQL checks the go-sql-driver/mysql DSN layout and, importantly, that
// credentials and database names are URL-query-escaped so that special
// characters in a password cannot corrupt the DSN.
func TestDSNMySQL(t *testing.T) {
	cfg := &Config{
		Dialect:  MySQL,
		Username: "team user",
		Password: "p@ss:w/rd",
		Database: "team db",
		Host:     "db.example.com",
		Port:     3306,
	}

	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("DSN(mysql): unexpected error %v", err)
	}

	// user:password@tcp(host:port)/db?params
	if !strings.Contains(dsn, "@tcp(db.example.com:3306)/") {
		t.Fatalf("mysql DSN missing tcp host section, got %q", dsn)
	}
	if !strings.Contains(dsn, url.QueryEscape("p@ss:w/rd")) {
		t.Fatalf("mysql DSN must URL-escape the password, got %q", dsn)
	}
	// The raw, unescaped password must not leak into the DSN.
	if strings.Contains(dsn, "p@ss:w/rd") {
		t.Fatalf("mysql DSN leaked the raw unescaped password, got %q", dsn)
	}
}

// TestDSNPostgres checks the key=value Postgres DSN layout and that all
// user-controlled fields are URL-escaped.
func TestDSNPostgres(t *testing.T) {
	cfg := &Config{
		Dialect:  Postgres,
		Username: "team user",
		Password: "p@ss word",
		Database: "team db",
		Host:     "db.example.com",
		Port:     5432,
	}

	dsn, err := cfg.DSN()
	if err != nil {
		t.Fatalf("DSN(postgres): unexpected error %v", err)
	}

	for _, want := range []string{
		"host=db.example.com",
		"port=5432",
		"user=" + url.QueryEscape("team user"),
		"password=" + url.QueryEscape("p@ss word"),
		"dbname=" + url.QueryEscape("team db"),
	} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("postgres DSN missing %q, got %q", want, dsn)
		}
	}
}

// TestDSNUnsupportedDialect ensures an unknown dialect is rejected with the
// sentinel error rather than producing a bogus connection string.
func TestDSNUnsupportedDialect(t *testing.T) {
	cfg := &Config{Dialect: "oracle"}

	dsn, err := cfg.DSN()
	if err == nil {
		t.Fatalf("DSN(unsupported): expected an error, got dsn %q", dsn)
	}
	if !errors.Is(err, ErrUnsupportedDialect) {
		t.Fatalf("DSN(unsupported): expected ErrUnsupportedDialect, got %v", err)
	}
	if dsn != "" {
		t.Fatalf("DSN(unsupported): expected empty dsn on error, got %q", dsn)
	}
}

// TestEncodeParams verifies params are deterministically URL-encoded (sorted by
// key) so DSNs are stable, and that an empty map yields an empty string.
func TestEncodeParams(t *testing.T) {
	if got := encodeParams(nil); got != "" {
		t.Fatalf("encodeParams(nil): expected empty, got %q", got)
	}

	got := encodeParams(map[string]string{"b": "2", "a": "1"})
	if got != "a=1&b=2" {
		t.Fatalf("encodeParams: expected deterministic sorted output a=1&b=2, got %q", got)
	}
}
