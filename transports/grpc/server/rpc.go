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
	ver := ts.server.GetVersion()

	return &proto.Version{
		Major:      ver.Major,
		Minor:      ver.Minor,
		Patch:      ver.Patch,
		Commit:     ver.Commit,
		Dirty:      ver.Dirty,
		CompiledAt: ver.CompiledAt,
		OS:         ver.OS,
		Arch:       ver.Arch,
	}, nil
}

// GetUsers returns the list of teamserver users and their status.
func (ts *rpcServer) GetUsers(context.Context, *proto.Empty) (*proto.Users, error) {
	users, err := ts.server.GetUsers()

	var userspb []*proto.User
	for _, user := range users {
		userspb = append(userspb, &proto.User{
			Name:   user.Name,
			Online: user.Online,
		})
	}

	return &proto.Users{Users: userspb}, err
}

// ClientLog accepts a stream of client logs to save on the teamserver.
func (ts *rpcServer) ClientLog(proto.Team_ClientLogServer) error {
	return status.Errorf(codes.Unimplemented, "method ClientLog not implemented")
}
