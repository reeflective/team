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
	"math/bits"
	"os"
	"path/filepath"
	"sync"

	"github.com/ncruces/go-sqlite3"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// sqliteRuntimeOnce guards the one-time global wazero runtime configuration.
var sqliteRuntimeOnce sync.Once

// configureSQLiteRuntime makes the pure-Go (wazero) SQLite engine reuse a
// persistent, on-disk cache of the compiled WASM module.
//
// Without it, wazero recompiles the ~1.5MB SQLite module on every process
// start (~2s on a typical machine), which makes short-lived invocations —
// shell completion in particular — painfully slow. With the cache, only the
// first run per wazero version pays that cost; subsequent runs load the
// precompiled module in tens of milliseconds.
//
// It is best-effort and never fatal: if the cache directory is unavailable, we
// leave ncruces' default runtime configuration in place (correct, just not
// cached). We also defer to any RuntimeConfig an embedding application may have
// set itself.
func configureSQLiteRuntime() {
	sqliteRuntimeOnce.Do(func() {
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

		// Match ncruces' default memory limit, which is otherwise skipped when a
		// custom RuntimeConfig is supplied: 256MB on 64-bit, 32MB on 32-bit.
		pages := uint32(4096)
		if bits.UintSize < 64 {
			pages = 512
		}

		// NewRuntimeConfig() selects the optimizing compiler where supported and
		// the interpreter otherwise; the cache is simply ignored by the latter.
		sqlite3.RuntimeConfig = wazero.NewRuntimeConfig().
			WithCompilationCache(cache).
			WithMemoryLimitPages(pages).
			WithCoreFeatures(api.CoreFeaturesV2)
	})
}
