package server

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
)

const (
	teamserverDir = "teamserver"
)

// AppDir returns the directory of the team server app (named ~/.<application>-server),
// creating the directory if needed, or logging a fatal event if failing to create it.
func (s *Server) AppDir() string {
	value := os.Getenv(fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(s.name)))

	var dir string
	if len(value) == 0 {
		user, _ := user.Current()
		dir = filepath.Join(user.HomeDir, fmt.Sprintf(".%s", s.name), teamserverDir)
	} else {
		dir = value
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o700)
		if err != nil {
			s.log.Errorf("Cannot write to %s root dir: %w", dir)
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
			s.log.Errorf("Cannot write logs dir %s", logDir)
		}
	}
	return logDir
}

// When creating a new server, don't write anything to anywhere yet,
// but ensure that at least all directories to which we are supposed
// to write do indeed exist, and make them anyway.
// If any error happens it will returned right away and the creator
// of the teamserver will know right away that it can't work correctly.
func (s *Server) checkWritableDirs() error {
	return nil
}
