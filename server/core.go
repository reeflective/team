package server

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
	"fmt"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/reeflective/team"
	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/internal/certs"
	"github.com/reeflective/team/internal/db"
	"github.com/reeflective/team/internal/version"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Server is a team server.
type Server struct {
	// Core
	name         string
	rootDirEnv   string
	fileLogger   *logrus.Logger
	stdoutLogger *logrus.Logger
	userTokens   *sync.Map
	initOnce     *sync.Once
	opts         *opts[any]
	db           *gorm.DB
	dbInitOnce   sync.Once
	certs        *certs.Manager
	fs           *assets.FS

	// Listeners and job control
	self     Handler[any]
	handlers map[string]Handler[any]
	jobs     *jobs
}

// New creates a new teamserver for the provided application name.
// This server can handle any number of remote clients for a given application
// named "teamserver", including its own local runtime (fully in-memory) client.
//
// All errors returned from this call are critical, in that the server could not
// run properly in its most basic state. If an error is raised, no server is returned.
//
// This call to create the server only creates the application default directory.
// No files, logs, connections or any interaction with the os/filesystem are made.
func New(application string, options ...Options) (*Server, error) {
	server := &Server{
		name:       application,
		rootDirEnv: fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(application)),
		opts:       newDefaultOpts(),

		userTokens: &sync.Map{},
		initOnce:   &sync.Once{},
		jobs:       newJobs(),
		handlers:   make(map[string]Handler[any]),
	}

	server.apply(options...)

	// Filesystem
	user, _ := user.Current()
	root := filepath.Join(user.HomeDir, "."+server.name)
	server.fs = assets.NewFileSystem(root, server.opts.inMemory)

	// Logging (if allowed)
	if err := server.initLogging(); err != nil {
		return nil, err
	}

	// Ensure we have a working database configuration,
	// and at least an in-memory sqlite database.
	if server.opts.dbConfig == nil {
		server.opts.dbConfig = server.getDefaultDatabaseConfig()
	}

	if server.opts.dbConfig.Database == db.SQLiteInMemoryHost && server.db == nil {
		if err := server.initDatabase(); err != nil {
			return nil, server.errorf("%w: %w", ErrDatabase, err)
		}
	}

	return server, nil
}

// Name returns the name of the application handled by the teamserver.
// Since you can embed multiple teamservers (one for each application)
// into a single binary, this is different from the program binary name
// running this teamserver.
func (ts *Server) Name() string {
	return ts.name
}

// Version returns the server binary version information.
func (ts *Server) Version() (team.Version, error) {
	dirty := version.GitDirty != ""
	semVer := version.Semantic()
	compiled, _ := version.Compiled()

	return team.Version{
		Major:      int32(semVer[0]),
		Minor:      int32(semVer[1]),
		Patch:      int32(semVer[2]),
		Commit:     strings.TrimSuffix(version.GitCommit, "\n"),
		Dirty:      dirty,
		CompiledAt: compiled.Unix(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}, nil
}

// Users returns the list of users in the teamserver database, and their information.
func (ts *Server) Users() ([]team.User, error) {
	if err := ts.initDatabase(); err != nil {
		return nil, ts.errorf("%w: %w", ErrDatabase, err)
	}

	usersDB := []*db.User{}
	err := ts.dbSession().Find(&usersDB).Error

	users := make([]team.User, len(usersDB))

	if err != nil && len(usersDB) == 0 {
		return users, ts.errorf("%w: %w", ErrDatabase, err)
	}

	for i, user := range usersDB {
		users[i] = team.User{
			Name:     user.Name,
			LastSeen: user.LastSeen,
		}

		if _, ok := ts.userTokens.Load(user.Token); ok {
			users[i].Online = true
		}
	}

	return users, nil
}

func (ts *Server) Filesystem() *assets.FS {
	return ts.fs
}

func (ts *Server) Self(opts ...client.Options) *client.Client {
	teamclient, _ := client.New(ts.Name(), ts, opts...)

	return teamclient
}
