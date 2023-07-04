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

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/server/certs"
	"github.com/reeflective/team/server/db"
)

// Server is a team server.
type Server struct {
	name       string
	rootDirEnv string
	listening  bool
	init       *sync.Once
	opts       *opts
	config     *Config
	db         *gorm.DB
	log        *logrus.Logger
	audit      *logrus.Logger
	certs      *certs.Manager
	*proto.UnimplementedTeamServer
}

// New creates a new team server with enabled classic/audit logging.
func New(application string, options ...Options) *Server {
	s := &Server{
		name:                    application,
		rootDirEnv:              fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(application)),
		init:                    &sync.Once{},
		opts:                    &opts{},
		UnimplementedTeamServer: &proto.UnimplementedTeamServer{},
	}

	// Logging
	s.log = s.rootLogger()
	s.audit = s.newAuditLogger()

	return s
}

// Name returns the name of the server application.
func (s *Server) Name() string {
	return s.name
}

// GracefulStop gracefully stops all components of the server,
// letting all current pending connections to it to finish first.
func (s *Server) GracefulStop() {
	defer s.log.Writer().Close()
	defer s.audit.Writer().Close()
}

func (s *Server) newServer() *Server {
	serv := &Server{
		name:                    s.name,
		rootDirEnv:              s.rootDirEnv,
		opts:                    s.opts,
		config:                  s.config,
		log:                     s.log,
		audit:                   s.audit,
		certs:                   s.certs,
		UnimplementedTeamServer: &proto.UnimplementedTeamServer{},
	}

	// One session per listener should be enough for now.
	serv.db = s.db.Session(&gorm.Session{
		FullSaveAssociations: true,
	})

	return serv
}

func (s *Server) Init(opts ...Options) {
	s.init.Do(func() {
		// Default and user options do not prevail
		// on what is in the configuration file
		s.apply(WithDatabaseConfig(s.GetDatabaseConfig()))
		s.apply(opts...)

		// Load any relevant server configuration: on disk,
		// contained in options, or the default one.
		s.config = s.GetConfig()

		// Database
		if s.opts.db == nil {
			s.db = db.NewDatabaseClient(s.opts.dbConfig, s.log)
		}

		// Certificate infrastructure
		certsLog := s.NamedLogger("certs", "certificates")
		s.certs = certs.NewManager(s.db.Session(&gorm.Session{}), certsLog, s.AppDir())
	})
}

// GetVersion returns the teamserver version.
func (s *Server) GetVersion(context.Context, *proto.Empty) (*proto.Version, error) {
	dirty := client.GitDirty != ""
	semVer := client.SemanticVersion()
	compiled, _ := client.Compiled()
	return &proto.Version{
		Major:      int32(semVer[0]),
		Minor:      int32(semVer[1]),
		Patch:      int32(semVer[2]),
		Commit:     strings.TrimSuffix(client.GitCommit, "\n"),
		Dirty:      dirty,
		CompiledAt: compiled.Unix(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}, nil
}

// ClientLog accepts a stream of client logs to save on the teamserver.
func (s *Server) ClientLog(proto.Team_ClientLogServer) error {
	return status.Errorf(codes.Unimplemented, "method ClientLog not implemented")
}

// GetUsers returns the list of teamserver users and their status.
func (s *Server) GetUsers(context.Context, *proto.Empty) (*proto.Users, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUsers not implemented")
}
