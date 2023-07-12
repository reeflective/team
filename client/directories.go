package client

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
	"os/user"
	"path/filepath"

	"github.com/reeflective/team/internal/log"
)

const (
	// configsDirName - Directory name containing config files.
	teamserverClientDir = "teamclient"
	configsDirName      = "configs"
	logFileExt          = "teamclient"
)

// AppDir returns the teamclient directory of the app (named ~/.<app>/teamserver/client/),
// creating the directory if needed, or logging an error event if failing to create it.
func (tc *Client) AppDir() string {
	user, _ := user.Current()
	dir := filepath.Join(user.HomeDir, "."+tc.name, teamserverClientDir)

	err := tc.fs.MkdirAll(dir, log.DirPerm)
	if err != nil {
		tc.log().Errorf("cannot write to %s root dir: %s", dir, err)
	}

	return dir
}

// LogsDir returns the directory of the client (~/.app/logs), creating
// the directory if needed, or logging a fatal event if failing to create it.
func (tc *Client) LogsDir() string {
	logsDir := filepath.Join(tc.AppDir(), "logs")

	err := tc.fs.MkdirAll(logsDir, log.DirPerm)
	if err != nil {
		tc.log().Errorf("cannot write to %s root dir: %s", logsDir, err)
	}

	return logsDir
}

// GetConfigDir - Returns the path to the config dir.
func (tc *Client) ConfigsDir() string {
	rootDir, _ := filepath.Abs(tc.AppDir())
	dir := filepath.Join(rootDir, configsDirName)

	err := tc.fs.MkdirAll(dir, log.DirPerm)
	if err != nil {
		tc.log().Errorf("cannot write to %s configs dir: %s", dir, err)
	}

	return dir
}
