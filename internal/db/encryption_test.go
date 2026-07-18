//go:build !cgo_sqlite

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
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// The adiantum encryption VFS is only available on the pure-Go builds (default
// and wasm_sqlite); the cgo_sqlite build uses a different SQLite engine, so this
// file is excluded there via the build constraint above.

const plaintextMarker = "ENCRYPTION_AT_REST_PLAINTEXT_MARKER"

func newTestDBConfig(path, key string) *Config {
	return &Config{
		Dialect:       Sqlite,
		Database:      path,
		MaxIdleConns:  1,
		MaxOpenConns:  1,
		LogLevel:      "error",
		EncryptionKey: key,
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// closeDB releases the underlying connection so the file is flushed and can be
// reopened / inspected.
func closeDB(t *testing.T, cfg *Config) {
	t.Helper()

	client, err := NewClient(cfg, discardLogger())
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.Create(&Certificate{CommonName: plaintextMarker}).Error; err != nil {
		t.Fatalf("insert marker: %v", err)
	}

	sqlDB, err := client.DB()
	if err != nil {
		t.Fatalf("client.DB: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

// TestEncryptedDatabaseAtRest is the end-to-end proof that WithDatabaseKey
// actually encrypts the database on disk: the file contains neither the
// plaintext marker nor the recognizable SQLite header, a wrong key cannot read
// it, and the correct key round-trips the data.
func TestEncryptedDatabaseAtRest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "encrypted.db")
	const key = "correct horse battery staple"

	// 1. Write with a key, then close.
	closeDB(t, newTestDBConfig(path, key))

	// 2. The on-disk file must be encrypted.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read db file: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("database file is empty")
	}
	if bytes.Contains(raw, []byte(plaintextMarker)) {
		t.Fatal("plaintext marker found in the on-disk database: not encrypted")
	}
	if bytes.HasPrefix(raw, []byte("SQLite format 3")) {
		t.Fatal("unencrypted SQLite header present in the on-disk database")
	}

	// 3. A wrong key must not be able to read the data.
	wrong, err := NewClient(newTestDBConfig(path, "wrong key"), discardLogger())
	if err == nil {
		var got Certificate
		if err := wrong.Where(&Certificate{CommonName: plaintextMarker}).First(&got).Error; err == nil {
			t.Fatal("a wrong key was able to decrypt and read the database")
		}
		if sqlDB, derr := wrong.DB(); derr == nil {
			sqlDB.Close()
		}
	}

	// 4. The correct key round-trips the data.
	right, err := NewClient(newTestDBConfig(path, key), discardLogger())
	if err != nil {
		t.Fatalf("NewClient(correct key): %v", err)
	}
	var got Certificate
	if err := right.Where(&Certificate{CommonName: plaintextMarker}).First(&got).Error; err != nil {
		t.Fatalf("correct key failed to read the marker back: %v", err)
	}
	if got.CommonName != plaintextMarker {
		t.Fatalf("round-trip mismatch: got %q", got.CommonName)
	}
}

// TestUnencryptedDatabaseIsPlaintext is the control: with no key, the same file
// is a normal SQLite database (opt-in encryption leaves the default untouched).
func TestUnencryptedDatabaseIsPlaintext(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.db")

	closeDB(t, newTestDBConfig(path, ""))

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read db file: %v", err)
	}
	if !bytes.HasPrefix(raw, []byte("SQLite format 3")) {
		t.Fatal("expected a standard SQLite header for an unencrypted database")
	}
}
