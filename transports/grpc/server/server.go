package server

import (
	"context"
	"net"
	"runtime/debug"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/test/bufconn"

	"github.com/reeflective/team"
	"github.com/reeflective/team/client"
	"github.com/reeflective/team/server"
	clientConn "github.com/reeflective/team/transports/grpc/client"
	"github.com/reeflective/team/transports/grpc/proto"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	bufSize = 2 * mb

	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = 2*gb - 1
)

type handler struct {
	*server.Server
	sconfig *server.Config

	options []grpc.ServerOption
	conn    *bufconn.Listener
	mutex   *sync.RWMutex
}

func NewTeamServer(opts ...grpc.ServerOption) *handler {
	h := &handler{
		mutex: &sync.RWMutex{},
	}

	// Buffering
	h.options = append(h.options,
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	)
	return h
}

// DialerFrom generates an in-memory, unauthenticated client dialer and server
func DialerFrom(server *handler) (teamclient team.Client, dialer client.Dialer[any]) {
	conn := bufconn.Listen(bufSize)

	ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return conn.Dial()
	})

	dialOpts := []grpc.DialOption{
		ctxDialer,
		grpc.WithInsecure(),
	}

	// The server will use this conn as a listener.
	// The reference is dropped after server start.
	server.conn = conn

	// Call the grpc client package for a dialer.
	return clientConn.NewTeamClient(dialOpts...)
}

// TeamClientFrom builds a complete teamclient from a server.
// It first generates the dialer and the teamclient with DialerFrom(server),
// then directly creates a new teamclient from the team/client package.
func TeamClientFrom(server *handler) (*client.Client, error) {
	teamclient, dialer := DialerFrom(server)

	return client.New(server.Name(), teamclient, client.WithDialer(dialer))
}

// Init implements server.Handler.Init(), and is used to initialize
// the server handler. Logging, connection options, anything can be
// done as long as it's for ensuring that the rest will work.
func (h *handler) Init(serv *server.Server) (err error) {
	h.Server = serv
	h.sconfig = h.Server.GetConfig()

	// Logging/authentication/audit
	serverOptions, err := h.initMiddleware()
	if err != nil {
		return err
	}

	h.options = append(h.options, serverOptions...)

	return nil
}

// Listen implements server.Handler.Listen().
// It starts listening on a network address for incoming gRPC clients.
// This connection CANNOT initiate in-memory connections.
func (h *handler) Listen(addr string) (net.Listener, error) {
	rpcLog := h.NamedLogger("transport", "mTLS")

	if h.conn != nil {
		return h.conn, nil
	}

	rpcLog.Debugf("Starting gRPC TLS listener on %s", addr)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return ln, nil
}

// Serve implements server.Handler.Serve().
// It accepts a network listener that will be served by a gRPC server.
// This also registers the Teamclient RPC service.
func (h *handler) Serve(ln net.Listener) (any, error) {
	rpcLog := h.NamedLogger("transport", "grpc")

	// Encryption.
	if h.conn == nil {
		rpcLog.Infof("Serving gRPC teamserver on %s", ln.Addr())

		tlsConfig := h.GetUserTLSConfig()
		creds := credentials.NewTLS(tlsConfig)
		h.options = append(h.options, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(h.options...)

	// If we already have an in-memory listener, use it.
	if h.conn != nil {
		ln = h.conn
		h.conn = nil
	}

	// Start serving the listener
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
	proto.RegisterTeamServer(grpcServer, newServer(h.Server))

	return grpcServer, nil
}

func (h *handler) Close() error {
	return nil
}
