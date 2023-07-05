package client

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

const (
	teamserverClientDir = "teamclient"
)

// AppDir returns the teamclient directory of the app (named ~/.<app>/teamserver/client/),
// creating the directory if needed, or logging an error event if failing to create it.
func (c *Client) AppDir() string {
	user, _ := user.Current()
	dir := filepath.Join(user.HomeDir, "."+c.name, teamserverClientDir)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o700)
		if err != nil {
			c.log.Errorf(fmt.Sprintf("cannot write to %s root dir: %w", dir, err))
		}
	}
	return dir
}

// LogsDir returns the directory of the client (~/.app/logs), creating
// the directory if needed, or logging a fatal event if failing to create it.
func (c *Client) LogsDir() string {
	logsDir := filepath.Join(c.AppDir(), "logs")
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		err = os.MkdirAll(logsDir, 0o700)
		if err != nil {
			c.log.Errorf(fmt.Sprintf("cannot write to %s root dir: %w", logsDir, err))
		}
	}
	return logsDir
}
