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

	"github.com/tetratelabs/wazero"
)

// buildRuntimeConfig returns an optimizing-compiler runtime backed by a
// persistent, on-disk cache of the compiled SQLite WASM module.
//
// Without it, wazero recompiles the ~1.5MB module on every process start
// (~2s), which makes short-lived invocations — shell completion in particular —
// painfully slow. With the cache, only the first run per wazero version pays
// that cost; later runs load the precompiled module in tens of milliseconds.
//
// It returns nil (leaving ncruces' default in place) when the cache directory
// is unavailable, so this is never fatal.
func buildRuntimeConfig() wazero.RuntimeConfig {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return nil
	}

	cacheDir := filepath.Join(cacheRoot, "reeflective-team", "sqlite-wasm")
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return nil
	}

	cache, err := wazero.NewCompilationCacheWithDir(cacheDir)
	if err != nil {
		return nil
	}

	// NewRuntimeConfig() selects the optimizing compiler where supported and the
	// interpreter otherwise; the cache is simply ignored by the latter.
	return wazero.NewRuntimeConfig().
		WithCompilationCache(cache).
		WithMemoryLimitPages(memoryLimitPages())
}
