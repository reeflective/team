//go:build !cgo_sqlite && race

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
	"github.com/ncruces/go-sqlite3"
	"github.com/tetratelabs/wazero"
)

// setupSQLiteRuntime uses wazero's pure-Go interpreter under the race detector.
//
// The optimizing compiler generates native machine code the Go race detector
// cannot instrument; under `-race` this occasionally traps with a spurious
// "wasm error: out of bounds memory access" during query execution. The
// interpreter is race-clean. Startup caching (and thus the cross-process cache
// lock) is irrelevant here, since this build is only produced by `go test
// -race`.
func setupSQLiteRuntime() {
	if sqlite3.RuntimeConfig != nil {
		return
	}

	sqlite3.RuntimeConfig = wazero.NewRuntimeConfigInterpreter().
		WithMemoryLimitPages(memoryLimitPages())
}
