package server

import (
	"context"

	"github.com/reeflective/team/server"
	"github.com/reeflective/team/transports/grpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type rpcServer struct {
	server *server.Server
	*proto.UnimplementedTeamServer
}

func newServer(server *server.Server) *rpcServer {
	return &rpcServer{
		server:                  server,
		UnimplementedTeamServer: &proto.UnimplementedTeamServer{},
	}
}

// GetVersion returns the teamserver version.
func (ts *rpcServer) GetVersion(context.Context, *proto.Empty) (*proto.Version, error) {
	ver, err := ts.server.Version()

	return &proto.Version{
		Major:      ver.Major,
		Minor:      ver.Minor,
		Patch:      ver.Patch,
		Commit:     ver.Commit,
		Dirty:      ver.Dirty,
		CompiledAt: ver.CompiledAt,
		OS:         ver.OS,
		Arch:       ver.Arch,
	}, err
}

// GetUsers returns the list of teamserver users and their status.
func (ts *rpcServer) GetUsers(context.Context, *proto.Empty) (*proto.Users, error) {
	users, err := ts.server.Users()

	userspb := make([]*proto.User, len(users))
	for i, user := range users {
		userspb[i] = &proto.User{
			Name:     user.Name,
			Online:   user.Online,
			LastSeen: user.LastSeen.Unix(),
			Clients:  int32(user.Clients),
		}
	}

	return &proto.Users{Users: userspb}, err
}

// ClientLog accepts a stream of client logs to save on the teamserver.
func (ts *rpcServer) ClientLog(proto.Team_ClientLogServer) error {
	return status.Errorf(codes.Unimplemented, "method ClientLog not implemented")
}
