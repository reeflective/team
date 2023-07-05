package client

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

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
func (c *Client) connect(config *Config) (*grpc.ClientConn, error) {
	tlsConfig, err := transport.GetTLSConfig(config.CACertificate, config.Certificate, config.PrivateKey)
	if err != nil {
		return nil, err
	}
	transportCreds := credentials.NewTLS(tlsConfig)
	callCreds := credentials.PerRPCCredentials(transport.TokenAuth(config.Token))
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithPerRPCCredentials(callCreds),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize)),
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	connection, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", config.Host, config.Port), options...)
	if err != nil {
		return nil, err
	}

	// Register the core RPC methods
	c.conn = connection
	c.rpc = proto.NewTeamClient(c.conn)

	return connection, nil
}
