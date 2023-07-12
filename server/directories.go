package server

import (
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
	logsDir       = "logs"
)

// AppDir returns the directory of the team server app (named ~/.<application>-server),
// creating the directory if needed, or logging a fatal event if failing to create it.
func (ts *Server) AppDir() string {
	value := os.Getenv(fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(ts.name)))

	var dir string

	if !ts.opts.inMemory {
		if len(value) == 0 {
			user, _ := user.Current()
			dir = filepath.Join(user.HomeDir, "."+ts.name, teamserverDir)
		} else {
			dir = value
		}
	} else {
		dir = filepath.Join("."+ts.name, teamserverDir)
	}

	err := ts.fs.MkdirAll(dir, log.DirPerm)
	if err != nil {
		ts.log().Errorf("cannot write to %s root dir: %s", dir, err)
	}

	return dir
}

// LogsDir returns the directory of the client (~/.app-server/logs), creating
// the directory if needed, or logging a fatal event if failing to create it.
func (ts *Server) LogsDir() string {
	rootDir := ts.AppDir()
	logDir := path.Join(rootDir, logsDir)

	err := ts.fs.MkdirAll(logDir, log.DirPerm)
	if err != nil {
		ts.log().Errorf("cannot write to %s root dir: %s", logDir, err)
	}

	return logDir
}

// When creating a new server, don't write anything to anywhere yet,
// but ensure that at least all directories to which we are supposed
// to write do indeed exist, and make them anyway.
// If any error happens it will returned right away and the creator
// of the teamserver will know right away that it can't work correctly.
func (ts *Server) checkWritableFiles() error {
	if ts.opts.inMemory {
		return nil
	}

	// Check home application directory.
	// If it does not exist but we don't have write permission
	// on /user/home, we return an error as we can't work.
	appDirWrite, err := log.IsWritable(ts.AppDir())

	switch {
	case os.IsNotExist(err):
		if homeWritable, err := log.IsWritable(os.Getenv("HOME")); !homeWritable {
			return fmt.Errorf("Cannot create %w", err)
		}
	case err != nil:
		return fmt.Errorf("Cannot write to %w", err)
	case !appDirWrite:
		return ErrDirectoryUnwritable
	}

	return nil
}
