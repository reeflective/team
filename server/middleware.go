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
	"github.com/reeflective/team/internal/log"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// initMiddleware - Initialize middleware logger
func (s *Server) initMiddleware() []grpc.ServerOption {
	logrusEntry := log.NamedLogger(s.log, "transport", "grpc")
	logrusOpts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(codeToLevel),
	}
	grpc_logrus.ReplaceGrpcLogger(logrusEntry)

	remoteAuth := !s.opts.local

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

func serverAuthFunc(ctx context.Context) (context.Context, error) {
	newCtx := context.WithValue(ctx, "transport", "local")
	newCtx = context.WithValue(newCtx, "user", "server")
	return newCtx, nil
}

func (s *Server) tokenAuthFunc(ctx context.Context) (context.Context, error) {
	mtlsLog := log.NamedLogger(s.log, "transport", "auth")
	mtlsLog.Debugf("Auth interceptor checking user token ...")
	rawToken, err := grpc_auth.AuthFromMD(ctx, "Bearer")
	if err != nil {
		mtlsLog.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}

	// Check auth cache
	digest := sha256.Sum256([]byte(rawToken))
	token := hex.EncodeToString(digest[:])
	newCtx := context.WithValue(ctx, "transport", "mtls")
	if name, ok := s.userTokens.Load(token); ok {
		mtlsLog.Debugf("Token in cache!")
		newCtx = context.WithValue(newCtx, "user", name.(string))
		return newCtx, nil
	}

	user, err := s.userByToken(token)
	if err != nil || user == nil {
		mtlsLog.Errorf("Authentication failure: %s", err)
		return nil, status.Error(codes.Unauthenticated, "Authentication failure")
	}
	mtlsLog.Debugf("Valid user token for %s", user.Name)
	s.userTokens.Store(token, user.Name)

	newCtx = context.WithValue(newCtx, "user", user.Name)
	return newCtx, nil
}

func (s *Server) deciderUnary(_ context.Context, _ string, _ interface{}) bool {
	return s.config.Log.GRPCUnaryPayloads
}

func (s *Server) deciderStream(_ context.Context, _ string, _ interface{}) bool {
	return s.config.Log.GRPCStreamPayloads
}

// Maps a grpc response code to a logging level
func codeToLevel(code codes.Code) logrus.Level {
	switch code {
	case codes.OK:
		return logrus.DebugLevel
	case codes.Canceled:
		return logrus.DebugLevel
	case codes.Unknown:
		return logrus.ErrorLevel
	case codes.InvalidArgument:
		return logrus.WarnLevel
	case codes.DeadlineExceeded:
		return logrus.WarnLevel
	case codes.NotFound:
		return logrus.DebugLevel
	case codes.AlreadyExists:
		return logrus.DebugLevel
	case codes.PermissionDenied:
		return logrus.WarnLevel
	case codes.Unauthenticated:
		return logrus.WarnLevel
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
}

func (s *Server) auditLogUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	log := log.NamedLogger(s.log, "transport", "middleware")

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
