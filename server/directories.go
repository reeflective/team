package server

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"github.com/reeflective/team/internal/log"
)

const (
	teamserverDir = "teamserver"
)

// AppDir returns the directory of the team server app (named ~/.<application>-server),
// creating the directory if needed, or logging a fatal event if failing to create it.
func (ts *Server) AppDir() string {
	value := os.Getenv(fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(ts.name)))

	var dir string
	if len(value) == 0 {
		user, _ := user.Current()
		dir = filepath.Join(user.HomeDir, fmt.Sprintf(".%s", ts.name), teamserverDir)
	} else {
		dir = value
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0o700)
		if err != nil {
			ts.log.Errorf("Cannot write to %s root dir: %w", dir)
		}
	}
	return dir
}

// LogsDir returns the directory of the client (~/.app-server/logs), creating
// the directory if needed, or logging a fatal event if failing to create it.
func (ts *Server) LogsDir() string {
	rootDir := ts.AppDir()
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		err = os.MkdirAll(rootDir, 0o700)
		if err != nil {
			ts.log.Errorf("Cannot write to %s root dir: %w", err)
		}
	}
	logDir := path.Join(rootDir, "logs")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err = os.MkdirAll(logDir, 0o700)
		if err != nil {
			ts.log.Errorf("Cannot write logs dir %s: %w", logDir, err)
		}
	}
	return logDir
}

// When creating a new server, don't write anything to anywhere yet,
// but ensure that at least all directories to which we are supposed
// to write do indeed exist, and make them anyway.
// If any error happens it will returned right away and the creator
// of the teamserver will know right away that it can't work correctly.
func (ts *Server) checkWritableFiles() error {
	// Check home application directory.
	// If it does not exist but we don't have write permission
	// on /user/home, we return an error as we can't work.
	appDirWrite, err := log.IsWritable(ts.AppDir())
	if err != nil && os.IsNotExist(err) {
		if homeWritable, err := log.IsWritable(os.Getenv("HOME")); !homeWritable {
			return fmt.Errorf("Cannot create application teamserver directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("Cannot write to application teamserver directory: %w", err)
	} else if !appDirWrite {
		return errors.New("The application teamserver directory seems to be unwritable")
	}

	return nil
}
