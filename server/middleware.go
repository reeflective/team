package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_tags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/reeflective/team/internal/log"
)

// initMiddleware - Initialize middleware logger
func (ts *Server) initMiddleware() []grpc.ServerOption {
	logrusEntry := log.NewNamed(ts.log, "transport", "grpc")
	logrusOpts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(log.CodeToLevel),
	}
	grpc_logrus.ReplaceGrpcLogger(logrusEntry)

	remoteAuth := !ts.opts.local

	if remoteAuth {
		return []grpc.ServerOption{
			grpc_middleware.WithUnaryServerChain(
				grpc_auth.UnaryServerInterceptor(ts.tokenAuthFunc),
				ts.auditLogUnaryServerInterceptor(),
				grpc_tags.UnaryServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
				grpc_logrus.UnaryServerInterceptor(logrusEntry, logrusOpts...),
				grpc_logrus.PayloadUnaryServerInterceptor(logrusEntry, ts.deciderUnary),
			),
			grpc_middleware.WithStreamServerChain(
				grpc_auth.StreamServerInterceptor(ts.tokenAuthFunc),
				grpc_tags.StreamServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
				grpc_logrus.StreamServerInterceptor(logrusEntry, logrusOpts...),
				grpc_logrus.PayloadStreamServerInterceptor(logrusEntry, ts.deciderStream),
			),
		}
	} else {
		return []grpc.ServerOption{
			grpc_middleware.WithUnaryServerChain(
				grpc_auth.UnaryServerInterceptor(serverAuthFunc),
				ts.auditLogUnaryServerInterceptor(),
				grpc_tags.UnaryServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
				grpc_logrus.UnaryServerInterceptor(logrusEntry, logrusOpts...),
				grpc_logrus.PayloadUnaryServerInterceptor(logrusEntry, ts.deciderUnary),
			),
			grpc_middleware.WithStreamServerChain(
				grpc_auth.StreamServerInterceptor(serverAuthFunc),
				grpc_tags.StreamServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
				grpc_logrus.StreamServerInterceptor(logrusEntry, logrusOpts...),
				grpc_logrus.PayloadStreamServerInterceptor(logrusEntry, ts.deciderStream),
			),
		}
	}
}

func serverAuthFunc(ctx context.Context) (context.Context, error) {
	newCtx := context.WithValue(ctx, "transport", "local")
	newCtx = context.WithValue(newCtx, "user", "server")
	return newCtx, nil
}

func (ts *Server) tokenAuthFunc(ctx context.Context) (context.Context, error) {
	log := log.NewNamed(ts.log, "transport", "auth")
	log.Debugf("Auth interceptor checking user token ...")
	rawToken, err := grpc_auth.AuthFromMD(ctx, "Bearer")
	if err != nil {
		log.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	// Check auth cache
	digest := sha256.Sum256([]byte(rawToken))
	token := hex.EncodeToString(digest[:])
	newCtx := context.WithValue(ctx, "transport", "mtls")
	if name, ok := ts.userTokens.Load(token); ok {
		log.Debugf("Token in cache!")
		newCtx = context.WithValue(newCtx, "user", name.(string))
		return newCtx, nil
	}

	user, err := ts.userByToken(token)
	if err != nil || user == nil {
		log.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}
	log.Debugf("Valid user token for %s", user.Name)
	ts.userTokens.Store(token, user.Name)

	newCtx = context.WithValue(newCtx, "user", user.Name)
	return newCtx, nil
}

func (ts *Server) deciderUnary(_ context.Context, _ string, _ interface{}) bool {
	return ts.config.Log.GRPCUnaryPayloads
}

func (ts *Server) deciderStream(_ context.Context, _ string, _ interface{}) bool {
	return ts.config.Log.GRPCStreamPayloads
}

type auditUnaryLogMsg struct {
	Request string `json:"request"`
	Method  string `json:"method"`
}

func (ts *Server) auditLogUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	log := log.NewNamed(ts.log, "transport", "middleware")

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
		ts.audit.Info(string(msgData))

		resp, err := handler(ctx, req)
		return resp, err
	}
}
