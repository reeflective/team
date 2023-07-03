package client

import (
	"fmt"
	"os"
	"path"

	"golang.org/x/exp/slog"
)

// Initialize logging
func (c *Client) initLogging(appDir string) *os.File {
	logFile, err := os.OpenFile(path.Join(appDir, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		panic(fmt.Sprintf("[!] Error opening file: %s", err))
	}

	// Initialize the log handler
	jsonOptions := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	jsonHandler := slog.NewJSONHandler(logFile, jsonOptions)
	c.log = slog.New(jsonHandler)

	return logFile
}
