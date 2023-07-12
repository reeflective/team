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
	"io/fs"
	"os"

	"github.com/psanford/memfs"
)

const (
	FileReadPerm  = 0o600 // FileReadPerm is the permission bit given to the OS when reading files.
	DirPerm       = 0o700 // DirPerm is the permission bit given to teamserver/client directories.
	FileWritePerm = 0o644 // FileWritePerm is the permission bit given to the OS when writing files.

	FileWriteOpenMode = os.O_APPEND | os.O_CREATE | os.O_WRONLY // Opening log files in append/create/write-only mode.
)

// FS is a filesystem abstraction for teamservers and teamclients.
// When either of them are configured to run in memory only, this
// filesystem is initialized accordingly, otherwise it will forward
// its calls to the on-disk filesystem.
type FS struct {
	mem *memfs.FS
}

// NewFileSystem returns a new filesystem configured to run on disk or in-memory.
func NewFileSystem(inMemory bool) *FS {
	filesystem := &FS{}

	if inMemory {
		filesystem.mem = memfs.New()
	}

	return filesystem
}

func (f *FS) MkdirAll(path string, perm fs.FileMode) error {
	if f.mem == nil {
		return os.MkdirAll(path, perm)
	}

	return f.mem.MkdirAll(path, perm)
}

func (f *FS) Sub(path string) (fs fs.FS, err error) {
	if f.mem == nil {
		return os.DirFS(path), nil
	}

	return f.mem.Sub(path)
}

func (f *FS) Open(name string) (fs.File, error) {
	if f.mem == nil {
		return os.Open(name)
	}

	return f.mem.Open(name)
}

func (f *FS) OpenFile(name string, flag int, perm fs.FileMode) (*File, error) {
	inFile := &File{
		name: name,
	}

	if f.mem == nil {
		file, err := os.OpenFile(name, flag, perm)
		if err != nil {
			return nil, err
		}

		inFile.file = file
	} else {
		inFile.mem = f.mem
	}

	return inFile, nil
}

func (f *FS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if f.mem == nil {
		return os.WriteFile(path, data, perm)
	}

	return f.mem.WriteFile(path, data, perm)
}

// File wraps the *os.File type with some in-memory helpers,
// so that we can write/read to teamserver application files
// regardless of where they are.
// This should disappear if a Write() method set is added to the io/fs package.
type File struct {
	name string
	file *os.File
	mem  *memfs.FS
}

// Write implements the io.Writer interface by writing data either
// to the file on disk, or to an in-memory file.
func (f *File) Write(data []byte) (written int, err error) {
	if f.file == nil {
		f.mem.WriteFile(f.name, data, FileWritePerm)
	}

	return f.file.Write(data)
}

// Close implements io.Closer by closing the file on the filesystem.
func (f *File) Close() error {
	if f.file == nil {
		return nil
	}

	return f.file.Close()
}
