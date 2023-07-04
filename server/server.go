package server

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/server/certs"
)

// Server is a team server.
type Server struct {
	name       string
	rootDirEnv string
	listening  bool
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
		opts:                    &opts{},
		UnimplementedTeamServer: &proto.UnimplementedTeamServer{},
	}

	// Logging
	s.log = s.rootLogger()
	s.audit = s.newAuditLogger()

	return s
}

// GracefulStop gracefully stops all components of the server,
// letting all current pending connections to it to finish first.
func (s *Server) GracefulStop() {
	defer s.log.Writer().Close()
	defer s.audit.Writer().Close()
}

// Name returns the name of the server application.
func (s *Server) Name() string {
	return s.name
}

func (s *Server) newServer() *Server {
	serv := &Server{
		name:       s.name,
		rootDirEnv: s.rootDirEnv,
		opts:       s.opts,
		config:     s.config,
		log:        s.log,
		audit:      s.audit,
	}

	// One session per listener should be enough for now.
	serv.db = s.db.Session(&gorm.Session{})

	// Certificate infrastructure
	// certsLog := s.NamedLogger("certs", "certificates")
	// serv.certs = certs.NewManager(serv.db, certsLog, s.AppDir())

	return serv
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
