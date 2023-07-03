package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/server/certs"
	"github.com/reeflective/team/server/db"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// Server is a team server.
type Server struct {
	name       string
	rootDirEnv string
	opts       *opts
	config     *ServerConfig
	db         *gorm.DB
	log        *logrus.Logger
	audit      *logrus.Logger
	certs      *certs.Manager
	rpc        *grpc.Server
	*proto.UnimplementedTeamServer
}

// NewServer creates a new team server with enabled classic/audit logging.
func NewServer(application string, options ...Options) *Server {
	s := &Server{
		name:                    application,
		rootDirEnv:              fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(application)),
		opts:                    &opts{},
		UnimplementedTeamServer: &proto.UnimplementedTeamServer{},
	}

	// Logging
	s.log = s.rootLogger()
	s.audit = s.newAuditLogger()

	// Default and user options
	s.apply(WithDatabaseConfig(s.GetDatabaseConfig()))
	s.apply(options...)

	// Database
	if s.opts.db == nil {
		s.db = db.NewDatabaseClient(s.opts.dbConfig, s.log)
	}

	// Certificate infrastructure
	certsLog := s.NamedLogger("certs", "certificates")
	s.certs = certs.NewManager(s.db, certsLog, s.AppDir())

	return s
}

// GetVersion returns the teamserver version.
func (s *Server) GetVersion(context.Context, *proto.Empty) (*proto.Version, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVersion not implemented")
}

// ClientLog accepts a stream of client logs to save on the teamserver.
func (s *Server) ClientLog(proto.Team_ClientLogServer) error {
	return status.Errorf(codes.Unimplemented, "method ClientLog not implemented")
}

// GetUsers returns the list of teamserver users and their status.
func (s *Server) GetUsers(context.Context, *proto.Empty) (*proto.Users, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUsers not implemented")
}
