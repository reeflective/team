package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_tags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// initMiddleware - Initialize middleware logger
func (s *Server) initMiddleware(remoteAuth bool) []grpc.ServerOption {
	logrusEntry := s.NamedLogger("transport", "grpc")
	logrusOpts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(codeToLevel),
	}
	grpc_logrus.ReplaceGrpcLogger(logrusEntry)

	if remoteAuth {
		return []grpc.ServerOption{
			grpc_middleware.WithUnaryServerChain(
				grpc_auth.UnaryServerInterceptor(s.tokenAuthFunc),
				s.auditLogUnaryServerInterceptor(),
				grpc_tags.UnaryServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
				grpc_logrus.UnaryServerInterceptor(logrusEntry, logrusOpts...),
				grpc_logrus.PayloadUnaryServerInterceptor(logrusEntry, s.deciderUnary),
			),
			grpc_middleware.WithStreamServerChain(
				grpc_auth.StreamServerInterceptor(s.tokenAuthFunc),
				grpc_tags.StreamServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
				grpc_logrus.StreamServerInterceptor(logrusEntry, logrusOpts...),
				grpc_logrus.PayloadStreamServerInterceptor(logrusEntry, s.deciderStream),
			),
		}
	} else {
		return []grpc.ServerOption{
			grpc_middleware.WithUnaryServerChain(
				grpc_auth.UnaryServerInterceptor(serverAuthFunc),
				s.auditLogUnaryServerInterceptor(),
				grpc_tags.UnaryServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
				grpc_logrus.UnaryServerInterceptor(logrusEntry, logrusOpts...),
				grpc_logrus.PayloadUnaryServerInterceptor(logrusEntry, s.deciderUnary),
			),
			grpc_middleware.WithStreamServerChain(
				grpc_auth.StreamServerInterceptor(serverAuthFunc),
				grpc_tags.StreamServerInterceptor(grpc_tags.WithFieldExtractor(grpc_tags.CodeGenRequestFieldExtractor)),
				grpc_logrus.StreamServerInterceptor(logrusEntry, logrusOpts...),
				grpc_logrus.PayloadStreamServerInterceptor(logrusEntry, s.deciderStream),
			),
		}
	}
}

var tokenCache = sync.Map{}

// clearTokenCache - Clear the auth token cache
func clearTokenCache() {
	tokenCache = sync.Map{}
}

func serverAuthFunc(ctx context.Context) (context.Context, error) {
	newCtx := context.WithValue(ctx, "transport", "local")
	newCtx = context.WithValue(newCtx, "operator", "server")
	return newCtx, nil
}

func (s *Server) tokenAuthFunc(ctx context.Context) (context.Context, error) {
	mtlsLog := s.NamedLogger("transport", "mtls")
	mtlsLog.Debugf("Auth interceptor checking operator token ...")
	rawToken, err := grpc_auth.AuthFromMD(ctx, "Bearer")
	if err != nil {
		mtlsLog.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	// Check auth cache
	digest := sha256.Sum256([]byte(rawToken))
	token := hex.EncodeToString(digest[:])
	newCtx := context.WithValue(ctx, "transport", "mtls")
	if name, ok := tokenCache.Load(token); ok {
		mtlsLog.Debugf("Token in cache!")
		newCtx = context.WithValue(newCtx, "operator", name.(string))
		return newCtx, nil
	}

	operator, err := s.UserByToken(token)
	if err != nil || operator == nil {
		mtlsLog.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}
	mtlsLog.Debugf("Valid user token for %s", operator.Name)
	tokenCache.Store(token, operator.Name)

	newCtx = context.WithValue(newCtx, "operator", operator.Name)
	return newCtx, nil
}

func (s *Server) deciderUnary(_ context.Context, _ string, _ interface{}) bool {
	return s.config.Logs.GRPCUnaryPayloads
}

func (s *Server) deciderStream(_ context.Context, _ string, _ interface{}) bool {
	return s.config.Logs.GRPCStreamPayloads
}

// Maps a grpc response code to a logging level
func codeToLevel(code codes.Code) logrus.Level {
	switch code {
	case codes.OK:
		return logrus.InfoLevel
	case codes.Canceled:
		return logrus.InfoLevel
	case codes.Unknown:
		return logrus.ErrorLevel
	case codes.InvalidArgument:
		return logrus.InfoLevel
	case codes.DeadlineExceeded:
		return logrus.WarnLevel
	case codes.NotFound:
		return logrus.InfoLevel
	case codes.AlreadyExists:
		return logrus.InfoLevel
	case codes.PermissionDenied:
		return logrus.WarnLevel
	case codes.Unauthenticated:
		return logrus.InfoLevel
	case codes.ResourceExhausted:
		return logrus.WarnLevel
	case codes.FailedPrecondition:
		return logrus.WarnLevel
	case codes.Aborted:
		return logrus.WarnLevel
	case codes.OutOfRange:
		return logrus.WarnLevel
	case codes.Unimplemented:
		return logrus.ErrorLevel
	case codes.Internal:
		return logrus.ErrorLevel
	case codes.Unavailable:
		return logrus.WarnLevel
	case codes.DataLoss:
		return logrus.ErrorLevel
	default:
		return logrus.ErrorLevel
	}
}

type auditUnaryLogMsg struct {
	Request string `json:"request"`
	Method  string `json:"method"`
	Session string `json:"session,omitempty"`
	Beacon  string `json:"beacon,omitempty"`
}

func (s *Server) auditLogUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	log := s.NamedLogger("transport", "middleware")

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
		s.audit.Info(string(msgData))

		resp, err := handler(ctx, req)
		return resp, err
	}
}
