package server

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/reeflective/team/internal/certs"
	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/internal/version"
	"github.com/reeflective/team/server/db"
)

// Server is a team server.
type Server struct {
	// Core
	name       string
	rootDirEnv string
	listening  bool
	log        *logrus.Logger
	audit      *logrus.Logger
	userTokens *sync.Map

	// Configurations
	opts   *opts
	config *Config
	db     *gorm.DB
	certs  *certs.Manager

	// Services
	init *sync.Once
	*proto.UnimplementedTeamServer
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
	var err error

	server := &Server{
		name:                    application,
		rootDirEnv:              fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(application)),
		userTokens:              &sync.Map{},
		opts:                    &opts{},
		init:                    &sync.Once{},
		config:                  getDefaultServerConfig(),
		UnimplementedTeamServer: &proto.UnimplementedTeamServer{},
	}

	// Ensure all teamserver-specific directories are writable.

	// Logging (not writing to files until init)
	level := logrus.Level(server.config.Log.Level)

	server.log, err = log.NewClient(server.LogsDir(), server.Name(), level)
	if err != nil {
		return nil, err
	}

	// Log all RPC requests and their content.
	if server.audit, err = log.NewAudit(server.AppDir()); err != nil {
		return nil, err
	}

	return server, nil
}

// GetVersion returns the teamserver version.
func (ts *Server) GetVersion(context.Context, *proto.Empty) (*proto.Version, error) {
	dirty := version.GitDirty != ""
	semVer := version.Semantic()
	compiled, _ := version.Compiled()
	return &proto.Version{
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

// GetUsers returns the list of teamserver users and their status.
func (ts *Server) GetUsers(context.Context, *proto.Empty) (*proto.Users, error) {
	users := []*db.User{}
	err := ts.db.Distinct("Name").Find(&users).Error

	var userspb *proto.Users
	for _, user := range users {
		userspb.Users = append(userspb.Users, &proto.User{
			Name: user.Name,
		})
	}

	return userspb, err
}

// ClientLog accepts a stream of client logs to save on the teamserver.
func (ts *Server) ClientLog(proto.Team_ClientLogServer) error {
	return status.Errorf(codes.Unimplemented, "method ClientLog not implemented")
}

// Name returns the name of the application handled by the teamserver.
// Since you can embed multiple teamservers (one for each application)
// into a single binary, this is different from the program binary name
// running this teamserver.
func (ts *Server) Name() string {
	return ts.name
}

func (ts *Server) newServer() *Server {
	serv := &Server{
		name:                    ts.name,
		rootDirEnv:              ts.rootDirEnv,
		log:                     ts.log,
		audit:                   ts.audit,
		opts:                    ts.opts,
		config:                  ts.config,
		certs:                   ts.certs,
		userTokens:              ts.userTokens,
		init:                    &sync.Once{},
		UnimplementedTeamServer: &proto.UnimplementedTeamServer{},
	}

	// One session per listener should be enough for now.
	serv.db = ts.db.Session(&gorm.Session{
		FullSaveAssociations: true,
	})

	return serv
}

func (ts *Server) initServer(opts ...Options) error {
	var err error

	ts.init.Do(func() {
		// Default and user options do not prevail
		// on what is in the configuration file
		ts.apply(WithDatabaseConfig(ts.GetDatabaseConfig()))
		ts.apply(opts...)

		// Load any relevant server configuration: on disk,
		// contained in options, or the default one.
		ts.config = ts.GetConfig()

		// Database
		if ts.opts.db == nil {
			ts.db, err = db.NewClient(ts.opts.dbConfig, ts.log)
			if err != nil {
				return
			}
		}

		// Certificate infrastructure
		certsLog := log.NewNamed(ts.log, "certs", "certificates")
		ts.certs = certs.NewManager(ts.db.Session(&gorm.Session{}), certsLog, ts.AppDir())
	})

	return err
}
