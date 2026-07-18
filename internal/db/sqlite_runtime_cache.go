//go:build !cgo_sqlite && !race

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
	"os"
	"path/filepath"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/tetratelabs/wazero"
)

const (
	// compileLockTimeout bounds how long we wait for another process to finish
	// the first (cold) compilation before proceeding anyway. Compilation itself
	// takes ~2s, so this is generously above it.
	compileLockTimeout = 20 * time.Second

	// compileLockStale is the age past which a lock file is assumed to have been
	// abandoned by a crashed process and may be stolen.
	compileLockStale = 30 * time.Second
)

// setupSQLiteRuntime configures the pure-Go SQLite engine to reuse a persistent,
// on-disk cache of the compiled WASM module.
//
// Without it, wazero recompiles the ~1.5MB module on every process start (~2s),
// which makes short-lived invocations — shell completion in particular —
// painfully slow. With the cache, only the first run per wazero version pays
// that cost; later runs load the precompiled module in tens of milliseconds.
//
// It is best-effort: if the cache directory is unavailable we leave ncruces'
// default in place, and we also defer to any RuntimeConfig an embedding
// application set itself.
func setupSQLiteRuntime() {
	if sqlite3.RuntimeConfig != nil {
		return
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return
	}

	cacheDir := filepath.Join(cacheRoot, "reeflective-team", "sqlite-wasm")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return
	}

	cache, err := wazero.NewCompilationCacheWithDir(cacheDir)
	if err != nil {
		return
	}

	// NewRuntimeConfig() selects the optimizing compiler where supported and the
	// interpreter otherwise; the cache is simply ignored by the latter.
	sqlite3.RuntimeConfig = wazero.NewRuntimeConfig().
		WithCompilationCache(cache).
		WithMemoryLimitPages(memoryLimitPages())

	warmCompilationCache(cacheDir)
}

// warmCompilationCache triggers the one-time module compilation while holding a
// cross-process advisory lock, so that concurrent cold starts don't race to
// write the shared cache. wazero populates the cache with a temp-file-then-
// rename, and on Windows that rename fails ("Access is denied") when another
// process holds the destination — which ncruces then turns into a fatal
// connection error. Serializing the cold compile avoids the collision; once the
// cache is warm, reads never conflict.
func warmCompilationCache(cacheDir string) {
	lockPath := filepath.Join(cacheDir, ".compile.lock")

	var lock *os.File

	deadline := time.Now().Add(compileLockTimeout)
	for time.Now().Before(deadline) {
		if f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600); err == nil {
			lock = f
			break
		}

		// Steal a lock left behind by a crashed process.
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > compileLockStale {
			os.Remove(lockPath)
			continue
		}

		time.Sleep(75 * time.Millisecond)
	}

	if lock != nil {
		defer func() {
			lock.Close()
			os.Remove(lockPath)
		}()
	}

	// As the lock holder we are the sole writer; if we timed out waiting, the
	// holder has by now populated the cache and this only reads it.
	_ = sqlite3.Initialize()
}
