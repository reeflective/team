package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/test/bufconn"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/server/certs"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = 2*gb - 1
)

const bufSize = 2 * mb

// ServeUser starts the main teamserver listener for the default system user:
// If the default application config file is found, and that we have determined
// that a sister server is running accordingly, we do NOT start the server, but
// instead connect as clients over to the teamserver, not using any database or
// server-only code in the process.
func (s *Server) Serve(cli *client.Client, opts ...Options) (*grpc.Server, error) {
	s.apply(opts...)

	// Client options
	var config *client.Config

	if s.opts.userDefault {
		config = cli.DefaultUserConfig()
	}

	// Initialize all backend things for this server:
	// database, certificate authorities and related loggers.
	s.init()

	var conn *grpc.ClientConn
	var server *grpc.Server
	var err error

	// If the default user configuration is the same as us,
	// or one of our multiplayer jobs, start our listeners
	// first and let the client connect afterwards.
	if !s.clientServerMatch(config) {
		s.opts.local = true
		conn, server, err = s.ServeLocal()
		if err != nil {
			return server, err
		}
	}

	// Attempt to connect with the user configuration.
	// Return if we are done, since we
	err = cli.Connect(client.WithConnection(conn))
	if err != nil {
		return server, err
	}

	return server, nil
}

// ServeLocal is used by any teamserver binary application to emulate the client-side
// functionality with itself. It returns a gRPC client connection to be registered to
// a client (team/client package), the gRPC server for registering per-application
// services, or an error if listening failed.
func (s *Server) ServeLocal() (*grpc.ClientConn, *grpc.Server, error) {
	bufConnLog := s.NamedLogger("transport", "local")
	bufConnLog.Infof("Binding gRPC to listener ...")

	ln := bufconn.Listen(bufSize)

	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}

	server, err := s.setupRPC(ln, options)

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

// ServeAddr sets and start a gRPC teamserver listener (MutualTLS), with registered core teamserver RPC services.
func (s *Server) ServeAddr(host string, port uint16) (*grpc.Server, net.Listener, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		s.log.Error(err)
		return nil, nil, err
	}

	server, err := s.ServeWith(ln)

	return server, ln, err
}

// ServeWith starts a gRPC teamserver on the provided listener (setting up MutualTLS on it).
func (s *Server) ServeWith(ln net.Listener) (*grpc.Server, error) {
	bufConnLog := s.NamedLogger("transport", "mtls")
	bufConnLog.Infof("Serving gRPC teamserver on %s", ln.Addr())

	tlsConfig := s.getOperatorServerTLSConfig("multiplayer")
	creds := credentials.NewTLS(tlsConfig)

	options := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}

	server, err := s.setupRPC(ln, options)

	return server, err
}

// setupRPC starts the gRPC server and register the core teamserver services to it.
func (s *Server) setupRPC(ln net.Listener, options []grpc.ServerOption) (*grpc.Server, error) {
	rpcLog := s.NamedLogger("transport", "rpc")

	options = append(options, s.initMiddleware(!s.opts.local)...)
	grpcServer := grpc.NewServer(options...)

	go func() {
		panicked := true
		defer func() {
			if panicked {
				rpcLog.Errorf("stacktrace from panic: %s", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			rpcLog.Warnf("gRPC server exited with error: %v", err)
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

// getOperatorServerTLSConfig - Generate the TLS configuration, we do now allow the end user
// to specify any TLS paramters, we choose sensible defaults instead
func (s *Server) getOperatorServerTLSConfig(host string) *tls.Config {
	caCertPtr, _, err := s.certs.GetUserCertificateAutority()
	if err != nil {
		s.log.Fatal("Failed to get users certificate authority")
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCertPtr)

	_, _, err = s.certs.UserServerGetCertificate(host)
	if err == certs.ErrCertDoesNotExist {
		s.certs.UserServerGenerateCertificate(host)
	}

	certPEM, keyPEM, err := s.certs.UserServerGetCertificate(host)
	if err != nil {
		s.log.Errorf("Failed to generate or fetch certificate %s", err)
		return nil
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		s.log.Fatalf("Error loading server certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	// if s.certs.TLSKeyLogger != nil {
	// 	tlsConfig.KeyLogWriter = s.certs.TLSKeyLogger
	// }

	return tlsConfig
}
