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
	"log/slog"
	"time"

	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_tags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/reeflective/team/example/transports/grpcslog/common"
	"github.com/reeflective/team/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

// BufferingOptions returns a list of server options with max send/receive
// message size, which value is that of the ServerMaxMessageSize variable (2GB).
func BufferingOptions() (options []grpc.ServerOption) {
	options = append(options,
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	)

	return
}

// LogMiddlewareOptions is a set of logging middleware options
// preconfigured to perform the following tasks:
// - Log all connections/disconnections to/from the teamserver listener.
// - Log all raw client requests into a teamserver audit file (see server.AuditLog()).
func LogMiddlewareOptions(serv *server.Server) ([]grpc.ServerOption, error) {
	var requestOpts []grpc.UnaryServerInterceptor
	var streamOpts []grpc.StreamServerInterceptor

	cfg := serv.GetConfig()

	// Audit-log all requests. Any failure to audit-log the requests
	// of this server will themselves be logged to the root teamserver log.
	auditLog, err := serv.AuditLogger()
	if err != nil {
		return nil, err
	}

	requestOpts = append(requestOpts, auditLogUnaryServerInterceptor(serv, auditLog))

	requestOpts = append(requestOpts,
		grpc_tags.UnaryServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
	)

	streamOpts = append(streamOpts,
		grpc_tags.StreamServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
	)

	// Logging interceptors: log the outcome of every call at a level derived
	// from its gRPC code, optionally dumping payloads when configured.
	logger := serv.NamedLogger("transport", "grpc")

	requestOpts = append(requestOpts,
		logUnaryServerInterceptor(logger, cfg.Log.GRPCUnaryPayloads),
	)

	streamOpts = append(streamOpts,
		logStreamServerInterceptor(logger, cfg.Log.GRPCStreamPayloads),
	)

	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(requestOpts...),
		grpc.ChainStreamInterceptor(streamOpts...),
	}, nil
}

// TLSAuthMiddlewareOptions is a set of transport security options which will use
// the preconfigured teamserver TLS (credentials) configuration to authenticate
// incoming client connections. The authentication is Mutual TLS, used because
// all teamclients will connect with a known TLS credentials set.
func TLSAuthMiddlewareOptions(s *server.Server) ([]grpc.ServerOption, error) {
	var options []grpc.ServerOption

	tlsConfig, err := s.UsersTLSConfig()
	if err != nil {
		return nil, err
	}

	creds := credentials.NewTLS(tlsConfig)
	options = append(options, grpc.Creds(creds))

	return options, nil
}

// initAuthMiddleware - Initialize middleware logger.
func (ts *Teamserver) initAuthMiddleware() ([]grpc.ServerOption, error) {
	var requestOpts []grpc.UnaryServerInterceptor
	var streamOpts []grpc.StreamServerInterceptor

	// Authentication interceptors.
	if ts.conn == nil {
		// All remote connections are users who need authentication.
		requestOpts = append(requestOpts,
			grpc_auth.UnaryServerInterceptor(ts.tokenAuthFunc),
		)

		streamOpts = append(streamOpts,
			grpc_auth.StreamServerInterceptor(ts.tokenAuthFunc),
		)
	} else {
		// Local in-memory connections have no auth.
		requestOpts = append(requestOpts,
			grpc_auth.UnaryServerInterceptor(serverAuthFunc),
		)

		streamOpts = append(streamOpts,
			grpc_auth.StreamServerInterceptor(serverAuthFunc),
		)
	}

	// Return middleware for all requests and stream interactions in gRPC.
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(requestOpts...),
		grpc.ChainStreamInterceptor(streamOpts...),
	}, nil
}

// ContextKey represents a gRPC context metadata key.
type ContextKey int

const (
	Transport ContextKey = iota
	User
)

func serverAuthFunc(ctx context.Context) (context.Context, error) {
	newCtx := context.WithValue(ctx, Transport, "local")
	newCtx = context.WithValue(newCtx, User, "server")

	return newCtx, nil
}

// tokenAuthFunc uses the core reeflective/team/server to authenticate user requests.
func (ts *Teamserver) tokenAuthFunc(ctx context.Context) (context.Context, error) {
	log := ts.NamedLogger("transport", "grpc")

	rawToken, err := grpc_auth.AuthFromMD(ctx, "Bearer")
	if err != nil {
		log.Error("Authentication failure", "error", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	// Authentication: let the core teamserver verify the token and tell us WHO
	// is calling. This is the only identity primitive the teamserver provides;
	// it carries no permissions/roles.
	user, err := ts.Authenticate(rawToken)
	if err != nil || user.Name == "" {
		log.Error("Authentication failure", "error", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	// Authorization/identity is the APPLICATION's job. This example simply
	// injects the authenticated team.User, but a real consumer would resolve
	// user.Name against its own model (roles, permissions, an Operator record)
	// and store its OWN identity type under its OWN context key here.
	newCtx := context.WithValue(ctx, Transport, user)
	newCtx = context.WithValue(newCtx, User, user)

	return newCtx, nil
}

type auditUnaryLogMsg struct {
	Request string `json:"request"`
	Method  string `json:"method"`
}

func auditLogUnaryServerInterceptor(ts *server.Server, auditLog *slog.Logger) grpc.UnaryServerInterceptor {
	log := ts.NamedLogger("grpc", "audit")

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		rawRequest, err := json.Marshal(req)
		if err != nil {
			log.Error("Failed to serialize request", "error", err)
			return handler(ctx, req)
		}

		log.Debug("Raw request", "payload", string(rawRequest))

		// Construct Log Message
		msg := &auditUnaryLogMsg{
			Request: string(rawRequest),
			Method:  info.FullMethod,
		}

		msgData, _ := json.Marshal(msg)
		auditLog.Info(string(msgData))

		return handler(ctx, req)
	}
}

// logUnaryServerInterceptor logs the outcome of each unary call at a level
// derived from its gRPC status code, optionally dumping the request payload.
func logUnaryServerInterceptor(logger *slog.Logger, logPayloads bool) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if logPayloads {
			if raw, err := json.Marshal(req); err == nil {
				logger.Debug("Received payload", "method", info.FullMethod, "payload", string(raw))
			}
		}

		start := time.Now()
		resp, err := handler(ctx, req)

		logger.Log(ctx, common.CodeToLevel(status.Code(err)), "unary call",
			"method", info.FullMethod, "duration", time.Since(start).String(), "error", err)

		return resp, err
	}
}

// logStreamServerInterceptor logs the outcome of each streaming call at a level
// derived from its gRPC status code.
func logStreamServerInterceptor(logger *slog.Logger, logPayloads bool) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		err := handler(srv, stream)

		logger.Log(stream.Context(), common.CodeToLevel(status.Code(err)), "stream call",
			"method", info.FullMethod, "duration", time.Since(start).String(), "error", err)

		return err
	}
}
