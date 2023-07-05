package server

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/test/bufconn"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/internal/proto"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = 2*gb - 1
)

const bufSize = 2 * mb

// ServeSelf returns a local runtime client to this teamserver.
func (s *Server) Self(opts ...Options) *client.Client {
	// Make a new default client: no working connection yet.
	cli := client.New(s.Name())

	// Get the connection strategy for this client:
	// Either prioritize finding a server, which and how.
	// if !s.clientServerMatch(cli.DefaultUserConfig()) {
	// }

	// Or serve a runtime local connection.
	return cli
}

// ServeUser starts the main teamserver listener for the default system user:
// If the default application config file is found, and that we have determined
// that a sister server is running accordingly, we do NOT start the server, but
// instead connect as clients over to the teamserver, not using any database or
// server-only code in the process.
func (s *Server) Serve(cli *client.Client, opts ...Options) (*grpc.Server, error) {
	// Initialize all backend things for this server:
	// database, certificate authorities and related loggers.
	s.initServer(opts...)

	// If the default user configuration is the same as us,
	// or one of our multiplayer jobs, start our listeners
	// first and let the client connect afterwards.
	conn, server, err := s.ServeLocal()
	if err != nil {
		return server, err
	}

	// Attempt to connect with the user configuration.
	// Return if we are done, since we
	err = cli.Connect(client.WithConnection(conn))
	if err != nil {
		return server, err
	}

	return server, nil
}

// ServeAddr sets and start a gRPC teamserver listener (on MutualTLS) with registered
// teamserver services onto it.
// Starting listeners from application code (not from teamserver' commands) should most
// of the time be done with this function, as it will return you the gRPC server to which
// you can attach any application-specific APIs.
func (s *Server) ServeAddr(host string, port uint16) (*grpc.Server, net.Listener, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		s.log.Error(err)
		return nil, nil, err
	}

	server, err := s.ServeWith(ln)

	return server, ln, err
}

// ServeLocal is used by any teamserver binary application to emulate the client-side
// functionality with itself. It returns a gRPC client connection to be registered to
// a client (team/client package), the gRPC server for registering per-application
// services, or an error if listening failed.
func (s *Server) ServeLocal(opts ...Options) (*grpc.ClientConn, *grpc.Server, error) {
	bufConnLog := log.NamedLogger(s.log, "transport", "local")
	bufConnLog.Infof("Binding gRPC to listener ...")

	// Initialize all backend things for this server:
	// database, certificate authorities and related loggers.
	s.initServer(opts...)
	s.opts.local = true

	ln := bufconn.Listen(bufSize)

	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}

	server, err := s.initRPC(ln, options)

	// And connect the client
	ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return ln.Dial()
	})

	dialOpts := []grpc.DialOption{
		ctxDialer,
		grpc.WithInsecure(), // This is an in-memory listener, no need for secure transport
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ServerMaxMessageSize)),
	}

	conn, err := grpc.DialContext(context.Background(), "bufnet", dialOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to dial bufnet: %s\n", err)
	}

	return conn, server, err
}

// ServeWith starts a gRPC teamserver on the provided listener (setting up MutualTLS on it).
func (s *Server) ServeWith(ln net.Listener, opts ...Options) (*grpc.Server, error) {
	bufConnLog := log.NamedLogger(s.log, "transport", "mtls")
	bufConnLog.Infof("Serving gRPC teamserver on %s", ln.Addr())

	// Initialize all backend things for this server:
	// database, certificate authorities and related loggers.
	s.initServer(opts...)

	tlsConfig := s.getUserTLSConfig("multiplayer")
	creds := credentials.NewTLS(tlsConfig)

	options := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}

	server, err := s.initRPC(ln, options)

	return server, err
}

// initRPC starts the gRPC server and register the core teamserver services to it.
func (s *Server) initRPC(ln net.Listener, options []grpc.ServerOption) (*grpc.Server, error) {
	rpcLog := log.NamedLogger(s.log, "transport", "rpc")

	options = append(options, s.initMiddleware()...)
	grpcServer := grpc.NewServer(options...)

	go func() {
		panicked := true
		defer func() {
			if panicked {
				rpcLog.Errorf("stacktrace from panic: %s", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			rpcLog.Errorf("gRPC server exited with error: %v", err)
		} else {
			panicked = false
		}
	}()

	// Register the core teamserver service
	proto.RegisterTeamServer(grpcServer, s.newServer())

	// Run user-specified hooks
	for _, postServeHook := range s.opts.preServeHooks {
		if err := postServeHook(s); err != nil {
			return grpcServer, err
		}
	}

	return grpcServer, nil
}
