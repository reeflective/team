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
	"gorm.io/gorm"

	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/server/certs"
	"github.com/reeflective/team/server/db"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = 2*gb - 1
)

const bufSize = 2 * mb

func (s *Server) Serve(opts ...Options) {
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

	// Default users
	if s.opts.userDefault {
	}
}

// ServeLocal is used by any teamserver binary application to emulate the client-side functionality with itself.
// It returns a gRPC client connection to be registered to a client (team/client package),
// the gRPC server for registering per-application services, or an error if listening failed.
func (s *Server) ServeLocal() (*grpc.ClientConn, *grpc.Server, error) {
	s.opts.local = true

	s.Serve()

	// Start the server.
	server, ln, err := s.serveLocal()

	// And connect the client
	ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return ln.Dial()
	})

	options := []grpc.DialOption{
		ctxDialer,
		grpc.WithInsecure(), // This is an in-memory listener, no need for secure transport
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ServerMaxMessageSize)),
	}

	conn, err := grpc.DialContext(context.Background(), "bufnet", options...)
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

// serveLocal - Bind gRPC server to an in-memory listener, which is
// typically used for unit testing, but ... it should be fine.
func (s *Server) serveLocal() (*grpc.Server, *bufconn.Listener, error) {
	bufConnLog := s.NamedLogger("transport", "local")
	bufConnLog.Infof("Binding gRPC to listener ...")

	ln := bufconn.Listen(bufSize)

	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}

	server, err := s.setupRPC(ln, options)

	return server, ln, err
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
