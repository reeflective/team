package server

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"net"
	"runtime/debug"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/reeflective/team"
	"github.com/reeflective/team/server"
	"github.com/reeflective/team/transports/grpc/proto"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	bufSize = 2 * mb

	// ServerMaxMessageSize is the server-side max gRPC message size (~2GB).
	ServerMaxMessageSize = 2*gb - 1
)

// Handler is a ready-to-use gRPC team/server.Handler (a "listener/server/RPC"
// transport stack). It is the supported, importable evolution of the code that
// used to live under example/transports/grpc, distilled from the production
// Sliver teamserver transport.
//
// The handler embeds a team/server.Server core and uses it for fetching
// server-side TLS credentials, authenticating users, audit/logging, and job
// control. Out of the box it provides, on every served listener:
//   - message buffering (2GB),
//   - panic recovery (a handler panic becomes codes.Internal, not a crash),
//   - audit logging of every request through the core AuditLogger(),
//   - Mutual-TLS transport credentials for remote listeners,
//   - token AUTHENTICATION for remote listeners (core Server.Authenticate),
//     injecting the resolved *team.User into the request context.
//
// It deliberately ships NO application services and NO authorization policy.
// Applications compose those in via:
//   - PostServe(hook): register your own gRPC services on the server.
//   - WithAuthorizer(a): install an authorization interceptor built on your
//     policy (team.Authorizer), consulted AFTER authentication resolves identity.
//
// A single Handler value serves both remote (real net.Listener + mTLS + auth)
// and in-memory (bufconn, no encryption/auth) connections; which one is decided
// by whether the handler was primed with an in-memory conn (see NewClientFrom).
type Handler struct {
	*server.Server

	options      []grpc.ServerOption
	conn         *bufconn.Listener
	mutex        *sync.RWMutex
	hooks        []func(*grpc.Server) error
	authorizer   team.Authorizer
	coreServices bool
}

// NewListener returns a gRPC teamserver handler loaded with the provided gRPC
// server options (message buffering is always added on top). By default the
// handler serves remote clients over TCP+mTLS. Register it with the teamserver
// via server.WithHandler().
func NewListener(opts ...grpc.ServerOption) *Handler {
	h := &Handler{
		mutex:   &sync.RWMutex{},
		options: BufferingOptions(),
	}

	h.options = append(h.options, opts...)

	return h
}

// NewClientFrom primes an existing gRPC Handler with an in-memory (bufconn)
// connection and returns the dial options a teamclient must use to reach it.
// The returned options set up a context dialer over the shared bufconn with TLS
// disabled: in-memory connections are trusted and neither encrypted nor
// authenticated. Pass the options to the transport's client dialer.
func NewClientFrom(h *Handler, opts ...grpc.DialOption) []grpc.DialOption {
	conn := bufconn.Listen(bufSize)

	ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return conn.Dial()
	})

	opts = append(opts,
		ctxDialer,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	// The server will use this conn as a listener.
	// The reference is dropped after server start.
	h.mutex.Lock()
	h.conn = conn
	h.mutex.Unlock()

	return opts
}

// PostServe registers one or more hooks to run against the *grpc.Server just
// before it starts serving, so applications can register their own gRPC
// services on it. Hooks run in registration order; a hook error aborts the
// listen. This is the primary seam for wiring application RPC onto the shared
// transport (the counterpart to the client dialer's PostDial/Conn()).
func (h *Handler) PostServe(hooks ...func(*grpc.Server) error) {
	h.hooks = append(h.hooks, hooks...)
}

// WithCoreServices registers the built-in teamserver Team service (users and
// version) on the served gRPC server, so a connected teamclient can query them
// remotely (e.g. the `teamserver client users` / version commands). It is
// opt-in: applications that expose their own users/version RPC (as Sliver does)
// leave it off. The transport's client dialer answers Users()/VersionServer()
// against this service.
func (h *Handler) WithCoreServices() {
	h.coreServices = true
}

// WithAuthorizer installs an application authorization policy. When set, remote
// listeners gain an authorization interceptor that, after authentication has
// resolved the caller identity, calls Authorize(user, fullMethod) and rejects
// the call with codes.PermissionDenied on a non-nil error. In-memory (local,
// trusted) connections bypass authorization. Passing nil is a no-op.
func (h *Handler) WithAuthorizer(a team.Authorizer) {
	h.authorizer = a
}

// Name implements team/server.Handler.Name(); the stack is keyed as "gRPC".
func (h *Handler) Name() string {
	return "gRPC"
}

// Init implements team/server.Handler.Init(). It binds the core teamserver and
// assembles the transport-agnostic middleware (buffering already set, plus
// logging/audit, recovery, and authentication). Transport-specific TLS
// credentials are added later in Listen(), only for remote listeners.
func (h *Handler) Init(serv *server.Server) (err error) {
	h.Server = serv

	// Logging/audit middleware (uses the core slog loggers).
	logOptions, err := h.logMiddlewareOptions()
	if err != nil {
		return err
	}

	h.options = append(h.options, logOptions...)

	// Recovery + authentication (+ authorization if set) middleware.
	authOptions := h.initAuthMiddleware()
	h.options = append(h.options, authOptions...)

	return nil
}

// Listen implements team/server.Handler.Listen(). For a remote listener it
// binds a TCP socket and adds Mutual-TLS credentials; for an in-memory listener
// it returns the primed bufconn unencrypted. In both cases it starts serving a
// gRPC server (with application services registered via PostServe) on the
// listener and returns immediately (non-blocking), as the Handler contract
// requires.
func (h *Handler) Listen(addr string) (ln net.Listener, err error) {
	h.mutex.RLock()
	inMemory := h.conn
	h.mutex.RUnlock()

	// In-memory connections are trusted: no TLS, no authentication.
	if inMemory == nil {
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			return nil, err
		}

		tlsOptions, err := TLSAuthMiddlewareOptions(h.Server)
		if err != nil {
			return nil, err
		}

		h.options = append(h.options, tlsOptions...)
	} else {
		h.mutex.Lock()
		ln = h.conn
		h.conn = nil
		h.mutex.Unlock()
	}

	h.ServeOn(ln)

	return ln, nil
}

// ServeOn builds the gRPC server (with the middleware assembled in Init() and
// the application services registered via PostServe) and serves it on an
// already-created listener, in the background. Handler.Listen() uses it for the
// default TCP/bufconn cases; it is exported so a custom transport that produces
// its own net.Listener (e.g. a Tailscale/tsnet listener) can embed this Handler
// and reuse the exact same server stack by calling ServeOn(ln) from its own
// Listen(). Init() MUST have run first.

// Close implements team/server.Handler.Close(). The underlying net.Listener is
// owned and closed by the core teamserver job control, and the per-listener
// gRPC server is stopped by serve() when Serve returns, so there is nothing to
// close here.
func (h *Handler) Close() error {
	return nil
}

func (h *Handler) ServeOn(ln net.Listener) {
	rpcLog := h.NamedLogger("transport", "grpc")

	grpcServer := grpc.NewServer(h.options...)

	// The built-in teamserver Team service (users/version), when enabled.
	if h.coreServices {
		proto.RegisterTeamServer(grpcServer, newCoreServer(h.Server))
	}

	// Let applications register their own gRPC services on the server.
	for _, hook := range h.hooks {
		if hook == nil {
			continue
		}

		if err := hook(grpcServer); err != nil {
			rpcLog.Error("service bind hook error", "error", err)
			return
		}
	}

	rpcLog.Info("Serving gRPC teamserver", "address", ln.Addr().String())

	go func() {
		panicked := true
		defer func() {
			if panicked {
				rpcLog.Error("stacktrace from panic", "stack", string(debug.Stack()))
			}
		}()

		if err := grpcServer.Serve(ln); err != nil {
			rpcLog.Error("gRPC server exited with error", "error", err)
		} else {
			panicked = false
		}

		// Serve returns once the listener is closed (e.g. via the core
		// ListenerClose, which closes the net.Listener). The team core never
		// stops the gRPC server itself, so without this the per-connection
		// handler goroutines would leak for the rest of the process every time
		// a listener is closed. Stop() releases them.
		grpcServer.Stop()
	}()
}

// compile-time guarantee that the handler satisfies the team server contract.
var _ server.Handler = (*Handler)(nil)
