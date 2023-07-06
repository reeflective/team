package client

import (
	"context"
	"fmt"
	"time"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/internal/transport"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ClientMaxReceiveMessageSize - Max gRPC message size ~2Gb
	ClientMaxReceiveMessageSize = (2 * gb) - 1 // 2Gb - 1 byte

	defaultTimeout = time.Duration(10 * time.Second)
)

// connect establishes a working gRPC client connection to the server specified in the configuration.
func (tc *Client) connect(config *Config) (*grpc.ClientConn, error) {
	// Cryptography
	tlsConfig, err := transport.GetTLSConfig(config.CACertificate, config.Certificate, config.PrivateKey)
	if err != nil {
		return nil, err
	}
	transportCreds := credentials.NewTLS(tlsConfig)
	callCreds := credentials.PerRPCCredentials(transport.TokenAuth(config.Token))

	// Logging
	logrusEntry := log.NewNamed(tc.log, "transport", "grpc")
	logrusOpts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(log.CodeToLevel),
	}
	grpc_logrus.ReplaceGrpcLogger(logrusEntry)

	// Assemble
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithPerRPCCredentials(callCreds),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize)),
		grpc.WithUnaryInterceptor(grpc_logrus.UnaryClientInterceptor(logrusEntry, logrusOpts...)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	connection, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", config.Host, config.Port), options...)
	if err != nil {
		return nil, err
	}

	// Register the core RPC methods
	tc.conn = connection
	tc.rpc = proto.NewTeamClient(tc.conn)

	return connection, nil
}
