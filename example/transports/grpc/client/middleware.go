package client

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

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/example/transports/grpc/common"
)

// TokenAuth extracts authentication metadata from contexts,
// specifically the "Authorization": "Bearer" key:value pair.
type TokenAuth string

// LogMiddlewareOptions is an example list of gRPC options with logging middleware set up.
// This transport uses its own logrus logger (common.Logrus) for the gRPC stack/requests
// events, independently of the core teamclient's slog loggers.
func LogMiddlewareOptions() []grpc.DialOption {
	// NOTE: we deliberately do NOT call grpc_logrus.ReplaceGrpcLogger here: it
	// mutates gRPC's process-global logger (grpclog.SetLoggerV2), which races
	// with running gRPC goroutines when dialers are (re)initialized.
	logrusEntry := common.LogEntry("transport", "grpc")
	logrusOpts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(common.CodeToLevel),
	}

	// Intercepting client requests.
	requestIntercept := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		rawRequest, err := json.Marshal(req)
		if err != nil {
			logrusEntry.Errorf("Failed to serialize: %s", err)
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		logrusEntry.Debugf("Raw request: %s", string(rawRequest))

		return invoker(ctx, method, req, reply, cc, opts...)
	}

	options := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithUnaryInterceptor(grpc_logrus.UnaryClientInterceptor(logrusEntry, logrusOpts...)),
		grpc.WithUnaryInterceptor(requestIntercept),
	}

	return options
}

func tlsAuthMiddleware(cli *client.Client) ([]grpc.DialOption, error) {
	config := cli.Config()
	if config.PrivateKey == "" {
		return nil, ErrNoTLSCredentials
	}

	tlsConfig, err := cli.NewTLSConfigFrom(config.CACertificate, config.Certificate, config.PrivateKey)
	if err != nil {
		return nil, err
	}

	transportCreds := credentials.NewTLS(tlsConfig)
	callCreds := credentials.PerRPCCredentials(TokenAuth(config.Token))

	return []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithPerRPCCredentials(callCreds),
	}, nil
}

// GetRequestMetadata return values that are mapped to request headers.
func (t TokenAuth) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": "Bearer " + string(t),
	}, nil
}

// RequireTransportSecurity always return true.
func (TokenAuth) RequireTransportSecurity() bool {
	return true
}
