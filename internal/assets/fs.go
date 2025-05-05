package assets

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

	"github.com/spf13/afero"
)

const (
	// FileReadPerm is the permission bit given to the OS when reading files.
	FileReadPerm = 0o600
	// DirPerm is the permission bit given to teamserver/client directories.
	DirPerm = 0o700
	// FileWritePerm is the permission bit given to the OS when writing files.
	FileWritePerm = 0o644

	// FileWriteOpenMode is used when opening log files in append/create/write-only mode.
	FileWriteOpenMode = os.O_APPEND | os.O_CREATE | os.O_WRONLY
)

const (
	// Teamclient.

	// DirClient is the name of the teamclient subdirectory.
	DirClient = "teamclient"
	// DirLogs subdirectory name.
	DirLogs = "logs"
	// DirConfigs subdirectory name.
	DirConfigs = "configs"

	// Teamserver.

	// DirServer is the name of the teamserver subdirectory.
	DirServer = "teamserver"
	// DirCerts subdirectory name.
	DirCerts = "certs"
)

// FS is a filesystem abstraction for teamservers and teamclients.
// When either of them are configured to run in memory only, this
// filesystem is initialized accordingly, otherwise it will forward
// its calls to the on-disk filesystem.
type FS = afero.Afero

// NewFileSystem returns a new filesystem
// configured to run on disk or in-memory.
func NewFileSystem(inMemory bool) *FS {
	if inMemory {
		return &afero.Afero{
			Fs: afero.NewMemMapFs(),
		}
	}

	return &afero.Afero{
		Fs: afero.NewOsFs(),
	}
}
