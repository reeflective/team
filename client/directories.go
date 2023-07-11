package client

import (
	"os"
	"os/user"
	"path/filepath"
)

const (
	// configsDirName - Directory name containing config files
	teamserverClientDir = "teamclient"
	configsDirName      = "configs"
	logFileExt          = "teamclient"
)

// AppDir returns the teamclient directory of the app (named ~/.<app>/teamserver/client/),
// creating the directory if needed, or logging an error event if failing to create it.
func (tc *Client) AppDir() string {
	user, _ := user.Current()
	dir := filepath.Join(user.HomeDir, "."+tc.name, teamserverClientDir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o700)
		if err != nil {
			tc.log().Errorf("cannot write to %s root dir: %w", dir, err)
		}
	}
	return dir
}

// LogsDir returns the directory of the client (~/.app/logs), creating
// the directory if needed, or logging a fatal event if failing to create it.
func (tc *Client) LogsDir() string {
	logsDir := filepath.Join(tc.AppDir(), "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		err = os.MkdirAll(logsDir, 0o700)
		if err != nil {
			tc.log().Errorf("cannot write to %s root dir: %w", logsDir, err)
		}
	}
	return logsDir
}

// GetConfigDir - Returns the path to the config dir.
func (tc *Client) ConfigsDir() string {
	rootDir, _ := filepath.Abs(tc.AppDir())
	dir := filepath.Join(rootDir, configsDirName)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o700)
		if err != nil {
			tc.log().Errorf("cannot write to %s configs dir: %w", dir, err)
		}
	}
	return dir
}
