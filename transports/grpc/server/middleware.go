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
	"encoding/json"
	"runtime/debug"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/reeflective/team"
	"github.com/reeflective/team/server"
)

// ContextKey is the type of the values this transport injects into a request
// context. Applications reading the authenticated identity should use the
// exported keys below.
type ContextKey int

const (
	// Transport is the context key under which the transport stores the raw
	// authenticated identity (a *team.User for remote calls, the string
	// "server" for in-memory calls).
	Transport ContextKey = iota

	// User is the context key under which the transport stores the
	// authenticated *team.User (nil-typed "server" identity for in-memory).
	// An application's own middleware may overwrite this with a richer,
	// app-specific identity resolved from user.Name.
	User
)

// BufferingOptions returns gRPC server options raising the max send/receive
// message size to ServerMaxMessageSize (~2GB).
func BufferingOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}
}

// TLSAuthMiddlewareOptions returns transport-security options that authenticate
// incoming client connections with the teamserver's Mutual-TLS configuration.
// All teamclients connect with a known client certificate, so the server can
// require and verify it.
func TLSAuthMiddlewareOptions(s *server.Server) ([]grpc.ServerOption, error) {
	tlsConfig, err := s.UsersTLSConfig()
	if err != nil {
		return nil, err
	}

	return []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsConfig))}, nil
}

// logMiddlewareOptions returns logging/audit interceptors backed by the core
// teamserver slog loggers: every request is recorded to the teamserver audit
// log (see server.AuditLogger()). Unlike the old example transport this keeps
// everything on slog and never touches gRPC's process-global logger (which is
// not concurrency-safe and races running servers).
func (h *Handler) logMiddlewareOptions() ([]grpc.ServerOption, error) {
	auditLog, err := h.AuditLogger()
	if err != nil {
		return nil, err
	}

	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(auditUnaryServerInterceptor(auditLog)),
	}, nil
}

// initAuthMiddleware assembles the recovery + authentication (+ authorization)
// interceptor chain. Recovery is always outermost. Remote listeners then
// authenticate every call and, if an authorizer is set, authorize it. In-memory
// listeners are trusted: they inject a synthetic "server" identity and skip
// authorization.
func (h *Handler) initAuthMiddleware() []grpc.ServerOption {
	var unary []grpc.UnaryServerInterceptor
	var stream []grpc.StreamServerInterceptor

	// Recovery is the OUTERMOST interceptor so it wraps everything below.
	unary = append(unary, recoveryUnaryServerInterceptor(h.NamedLogger("transport", "grpc")))
	stream = append(stream, recoveryStreamServerInterceptor(h.NamedLogger("transport", "grpc")))

	if h.conn == nil {
		// Remote connections: authenticate identity first...
		unary = append(unary, grpc_auth.UnaryServerInterceptor(h.tokenAuthFunc))
		stream = append(stream, grpc_auth.StreamServerInterceptor(h.tokenAuthFunc))

		// ...then authorize, if the application supplied a policy. Order
		// matters: the authorizer reads the identity the auth step resolves.
		if h.authorizer != nil {
			unary = append(unary, h.authorizeUnaryServerInterceptor())
			stream = append(stream, h.authorizeStreamServerInterceptor())
		}
	} else {
		// Local in-memory connections are trusted (no auth, no authz).
		unary = append(unary, grpc_auth.UnaryServerInterceptor(serverAuthFunc))
		stream = append(stream, grpc_auth.StreamServerInterceptor(serverAuthFunc))
	}

	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unary...),
		grpc.ChainStreamInterceptor(stream...),
	}
}

// serverAuthFunc is the local, in-memory path: no authentication, a synthetic
// "server" identity injected so downstream handlers see a consistent context
// shape whether the call came in-memory or over the wire.
func serverAuthFunc(ctx context.Context) (context.Context, error) {
	ctx = context.WithValue(ctx, Transport, "server")
	ctx = context.WithValue(ctx, User, (*team.User)(nil))

	return ctx, nil
}

// tokenAuthFunc authenticates a remote call: it extracts the Bearer token and
// asks the core teamserver WHO is calling. The teamserver only proves identity
// (a name); it carries no permissions. Authorization is the application's job,
// via WithAuthorizer / the injected *team.User.
func (h *Handler) tokenAuthFunc(ctx context.Context) (context.Context, error) {
	log := h.NamedLogger("transport", "grpc")

	rawToken, err := grpc_auth.AuthFromMD(ctx, "Bearer")
	if err != nil {
		log.Error("Authentication failure", "error", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	user, err := h.Authenticate(rawToken)
	if err != nil || user == nil || user.Name == "" {
		log.Error("Authentication failure", "error", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	ctx = context.WithValue(ctx, Transport, user)
	ctx = context.WithValue(ctx, User, user)

	return ctx, nil
}

// authorizeUnaryServerInterceptor enforces the application authorization policy
// on unary calls, using the identity resolved by tokenAuthFunc and the full RPC
// method name as the action.
func (h *Handler) authorizeUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	log := h.NamedLogger("transport", "authz")

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		user, ok := ctx.Value(User).(*team.User)
		if !ok || user == nil || user.Name == "" {
			return nil, status.Error(codes.Unauthenticated, "Authentication failure")
		}

		if err := h.authorizer.Authorize(user.Name, info.FullMethod); err != nil {
			log.Warn("Permission denied", "user", user.Name, "method", info.FullMethod, "error", err)
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}

		return handler(ctx, req)
	}
}

// authorizeStreamServerInterceptor is the streaming counterpart of
// authorizeUnaryServerInterceptor.
func (h *Handler) authorizeStreamServerInterceptor() grpc.StreamServerInterceptor {
	log := h.NamedLogger("transport", "authz")

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		user, ok := ss.Context().Value(User).(*team.User)
		if !ok || user == nil || user.Name == "" {
			return status.Error(codes.Unauthenticated, "Authentication failure")
		}

		if err := h.authorizer.Authorize(user.Name, info.FullMethod); err != nil {
			log.Warn("Permission denied", "user", user.Name, "method", info.FullMethod, "error", err)
			return status.Error(codes.PermissionDenied, err.Error())
		}

		return handler(srv, ss)
	}
}

// recoveryUnaryServerInterceptor converts a panic in any downstream interceptor
// or unary handler into a codes.Internal error (logging method + stack) instead
// of letting it unwind through gRPC and crash the whole teamserver.
func recoveryUnaryServerInterceptor(log logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered", "method", info.FullMethod, "panic", r, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// recoveryStreamServerInterceptor is the streaming counterpart of
// recoveryUnaryServerInterceptor.
func recoveryStreamServerInterceptor(log logger) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("panic recovered", "method", info.FullMethod, "panic", r, "stack", string(debug.Stack()))
				err = status.Error(codes.Internal, "internal server error")
			}
		}()

		return handler(srv, ss)
	}
}

// auditUnaryServerInterceptor records the raw request and method of every unary
// call to the teamserver audit log.
func auditUnaryServerInterceptor(auditLog logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		rawRequest, err := json.Marshal(req)
		if err == nil {
			msg, _ := json.Marshal(struct {
				Request string `json:"request"`
				Method  string `json:"method"`
			}{Request: string(rawRequest), Method: info.FullMethod})
			auditLog.Info(string(msg))
		}

		return handler(ctx, req)
	}
}

// logger is the minimal slog surface the interceptors need, satisfied by
// *slog.Logger (from the core NamedLogger()/AuditLogger()).
type logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}
