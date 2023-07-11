package server

import (
	"context"
	"encoding/json"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_tags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/transports/grpc/common"
)

// initMiddleware - Initialize middleware logger
func (ts *handler) initMiddleware() ([]grpc.ServerOption, error) {
	var requestOpts []grpc.UnaryServerInterceptor
	var streamOpts []grpc.StreamServerInterceptor

	// Audit-log all requests. Any failure to audit-log the requests
	// of this server will themselves be logged to the root teamserver log.
	auditLog, err := log.NewAudit(ts.LogsDir())
	if err != nil {
		return nil, err
	}

	requestOpts = append(requestOpts, ts.auditLogUnaryServerInterceptor(auditLog))

	requestOpts = append(requestOpts,
		grpc_tags.UnaryServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
	)
	streamOpts = append(streamOpts,
		grpc_tags.StreamServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
	)

	// Logging interceptors
	logrusEntry := ts.NamedLogger("transport", "grpc")
	logrusOpts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(common.CodeToLevel),
	}
	grpc_logrus.ReplaceGrpcLogger(logrusEntry)

	requestOpts = append(requestOpts,
		grpc_logrus.UnaryServerInterceptor(logrusEntry, logrusOpts...),
		grpc_logrus.PayloadUnaryServerInterceptor(logrusEntry, ts.deciderUnary),
	)
	streamOpts = append(streamOpts,
		grpc_logrus.StreamServerInterceptor(logrusEntry, logrusOpts...),
		grpc_logrus.PayloadStreamServerInterceptor(logrusEntry, ts.deciderStream),
	)

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
		grpc_middleware.WithUnaryServerChain(requestOpts...),
		grpc_middleware.WithStreamServerChain(streamOpts...),
	}, nil
}

// TODO: Should we change the default in-memory server name ?
func serverAuthFunc(ctx context.Context) (context.Context, error) {
	newCtx := context.WithValue(ctx, "transport", "local")
	newCtx = context.WithValue(newCtx, "user", "server")
	return newCtx, nil
}

func (ts *handler) tokenAuthFunc(ctx context.Context) (context.Context, error) {
	log := ts.NamedLogger("transport", "grpc")
	log.Debugf("Auth interceptor checking user token ...")

	rawToken, err := grpc_auth.AuthFromMD(ctx, "Bearer")
	if err != nil {
		log.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	user, authorized, err := ts.AuthenticateUser(rawToken)
	if err != nil || !authorized || user == "" {
		log.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	newCtx := context.WithValue(ctx, "transport", "mtls")
	newCtx = context.WithValue(newCtx, "user", user)

	return newCtx, nil
}

func (ts *handler) deciderUnary(_ context.Context, _ string, _ interface{}) bool {
	return ts.sconfig.Log.GRPCUnaryPayloads
}

func (ts *handler) deciderStream(_ context.Context, _ string, _ interface{}) bool {
	return ts.sconfig.Log.GRPCStreamPayloads
}

type auditUnaryLogMsg struct {
	Request string `json:"request"`
	Method  string `json:"method"`
}

func (ts *handler) auditLogUnaryServerInterceptor(auditLog *logrus.Logger) grpc.UnaryServerInterceptor {
	log := ts.NamedLogger("grpc", "audit")

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ interface{}, err error) {
		rawRequest, err := json.Marshal(req)
		if err != nil {
			log.Errorf("Failed to serialize %s", err)
			return
		}

		log.Debugf("Raw request: %s", string(rawRequest))

		if err != nil {
			log.Errorf("Middleware failed to insert details: %s", err)
		}

		// Construct Log Message
		msg := &auditUnaryLogMsg{
			Request: string(rawRequest),
			Method:  info.FullMethod,
		}

		msgData, _ := json.Marshal(msg)
		auditLog.Info(ctx, string(msgData))

		resp, err := handler(ctx, req)
		return resp, err
	}
}
