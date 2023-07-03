package client

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
)

// AppDir returns the directory of the team application (named ~/.<application>),
// creating the directory if needed, or logging a fatal event if failing to create it.
func (c *Client) AppDir() string {
	user, _ := user.Current()
	dir := filepath.Join(user.HomeDir, c.name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o700)
		if err != nil {
			log.Fatal(err)
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
			log.Fatal(err)
		}
	}
	return logsDir
}

// GetConsoleLogsDir - Get the client console logs dir ~/.app/logs/console/
func (c *Client) ConsoleLogsDir() string {
	consoleLogsDir := filepath.Join(c.LogsDir(), "console")
	if _, err := os.Stat(consoleLogsDir); os.IsNotExist(err) {
		err = os.MkdirAll(consoleLogsDir, 0o700)
		if err != nil {
			log.Fatal(err)
		}
	}
	return consoleLogsDir
}
