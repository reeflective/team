//go:build !cgo_sqlite

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
	"bytes"
	"os"
	"testing"
)

// TestDatabaseEncryptionAtRest exercises the public WithDatabaseKey option end
// to end: a file-based teamserver created with a key must produce an on-disk
// database that is encrypted (no SQLite header, no user data in the clear). The
// adiantum VFS used here is pure-Go, so this test excludes the cgo_sqlite build.
func TestDatabaseEncryptionAtRest(t *testing.T) {
	const userMarker = "ENCRYPTED_USER_MARKER"

	home := t.TempDir()

	// WithNoLogs keeps the database on disk (unlike WithInMemory) while avoiding
	// an open log-file handle, which on Windows would block the t.TempDir()
	// cleanup ("the process cannot access the file because it is being used by
	// another process").
	ts, err := New("enctest",
		WithHomeDirectory(home),
		WithNoLogs(true),
		WithDatabaseKey("correct horse battery staple"),
	)
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	if err := ts.init(); err != nil {
		t.Fatalf("server.init: %v", err)
	}

	// Write some recognizable data into the database.
	if _, err := ts.UserCreate(userMarker, "localhost", 31337); err != nil {
		t.Fatalf("UserCreate: %v", err)
	}

	// Resolve the on-disk database path and flush the connection.
	dbPath := ts.opts.dbConfig.Database
	if dbPath == "" {
		t.Fatal("expected a file-based database path")
	}
	if sqlDB, derr := ts.db.DB(); derr == nil {
		sqlDB.Close()
	}

	raw, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read database file %q: %v", dbPath, err)
	}
	if len(raw) == 0 {
		t.Fatal("database file is empty")
	}
	if bytes.HasPrefix(raw, []byte("SQLite format 3")) {
		t.Fatal("database is not encrypted: standard SQLite header present")
	}
	if bytes.Contains(raw, []byte(userMarker)) {
		t.Fatal("database is not encrypted: user name found in cleartext on disk")
	}
}
