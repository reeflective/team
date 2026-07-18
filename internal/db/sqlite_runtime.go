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
	"sync"
)

// sqliteRuntimeOnce guards the one-time global wazero runtime configuration.
var sqliteRuntimeOnce sync.Once

// configureSQLiteRuntime installs a global wazero runtime configuration for the
// pure-Go SQLite engine, once, before the first connection is opened. The actual
// setup (optimizing compiler + persistent module cache in normal builds, or the
// interpreter under the race detector) is provided by setupSQLiteRuntime in a
// build-tagged file. It is best-effort: on any failure ncruces' own default is
// left in place.
func configureSQLiteRuntime() {
	sqliteRuntimeOnce.Do(setupSQLiteRuntime)
}

// memoryLimitPages mirrors ncruces' default WASM memory limit, which is skipped
// when we supply our own RuntimeConfig: 256MB on 64-bit, 32MB on 32-bit.
func memoryLimitPages() uint32 {
	if bits.UintSize < 64 {
		return 512
	}

	return 4096
}
