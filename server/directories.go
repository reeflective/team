package server

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
)

// AppDir returns the directory of the team server app (named ~/.<application>-server),
// creating the directory if needed, or logging a fatal event if failing to create it.
func (s *Server) AppDir() string {
	value := os.Getenv(fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(s.name)))

	var dir string
	if len(value) == 0 {
		user, _ := user.Current()
		dir = filepath.Join(user.HomeDir, fmt.Sprintf(".%s-server", s.name))
	} else {
		dir = value
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o700)
		if err != nil {
			msg := fmt.Sprintf("Cannot write to %s root dir", dir)
			panic(msg)
		}
	}
	return dir
}

// LogsDir returns the directory of the client (~/.app-server/logs), creating
// the directory if needed, or logging a fatal event if failing to create it.
func (s *Server) LogsDir() string {
	rootDir := s.AppDir()
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		err = os.MkdirAll(rootDir, 0o700)
		if err != nil {
			panic(err)
		}
	}
	logDir := path.Join(rootDir, "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err = os.MkdirAll(logDir, 0o700)
		if err != nil {
			panic(err)
		}
	}
	return logDir
}
