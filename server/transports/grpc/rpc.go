package grpc

import (
	"context"
	"runtime"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/internal/version"
	"github.com/reeflective/team/server"
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
func (ts *rpcServer) GetUsers(context.Context, *proto.Empty) (*proto.Users, error) {
	userspb, err := ts.server.GetUsers()
	return &proto.Users{Users: userspb}, err
}

// ClientLog accepts a stream of client logs to save on the teamserver.
func (ts *rpcServer) ClientLog(proto.Team_ClientLogServer) error {
	return status.Errorf(codes.Unimplemented, "method ClientLog not implemented")
}
