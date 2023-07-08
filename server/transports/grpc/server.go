package grpc

import (
	"context"
	"net"
	"net/url"
	"runtime/debug"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/test/bufconn"

	"github.com/reeflective/team/client"
	clientConn "github.com/reeflective/team/client/transports/grpc"
	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/server"
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

	options []grpc.ServerOption
	conn    *bufconn.Listener
	mutex   *sync.RWMutex
}

func NewServer(opts ...grpc.ServerOption) *handler {
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
func DialerFrom(server *handler) (teamclient client.Teamclient[any]) {
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

// Init implements server.Handler.Init(), and is used to initialize
// the server handler. Logging, connection options, anything can be
// done as long as it's for ensuring that the rest will work.
func (h *handler) Init(serv *server.Server) (err error) {
	h.Server = serv

	// Logging/authentication/audit
	serverOptions, err := h.initMiddleware()
	if err != nil {
		return err
	}

	h.options = append(h.options, serverOptions...)

	return nil
}

func (h *handler) Listen(addr string) (net.Listener, error) {
	if h.conn != nil {
		return h.conn, nil
	}

	// Parse the address into a URL.
	// We just want to keep the host:port combination.
	url, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	ln, err := net.Listen("tcp", url.Host)
	if err != nil {
		return nil, err
	}

	return ln, nil
}

func (h *handler) Serve(ln net.Listener) (any, error) {
	rpcLog := h.NamedLogger("transport", "grpc")
	rpcLog.Infof("Serving gRPC teamserver on %s", ln.Addr())

	// Encryption.
	if h.conn == nil {
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
