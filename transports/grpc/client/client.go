package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	"github.com/reeflective/team"
	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/internal/transport"
	"github.com/reeflective/team/transports/grpc/proto"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ClientMaxReceiveMessageSize - Max gRPC message size ~2Gb
	ClientMaxReceiveMessageSize = (2 * gb) - 1 // 2Gb - 1 byte

	defaultTimeout = time.Duration(10 * time.Second)
)

type handler struct {
	*client.Client
	target  string
	rpc     proto.TeamClient
	options []grpc.DialOption
}

func NewTeamClient(opts ...grpc.DialOption) (tc team.Client, dialer client.Dialer[any]) {
	h := &handler{
		options: opts,
	}

	return h, h
}

func (h *handler) Init(cli *client.Client) error {
	h.Client = cli
	config := cli.Config()

	// Logging
	logrusEntry := cli.NamedLogger("transport", "grpc")
	logrusOpts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(log.CodeToLevel),
	}
	grpc_logrus.ReplaceGrpcLogger(logrusEntry)

	options := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize)),
		grpc.WithUnaryInterceptor(grpc_logrus.UnaryClientInterceptor(logrusEntry, logrusOpts...)),
		grpc.WithUnaryInterceptor(h.loggingInterceptor(logrusEntry)),
	}

	// If the configuration has no credentials, we are most probably
	// an in-memory dialer, don't authenticate and encrypt the conn.
	if config.PrivateKey != "" {
		tlsConfig, err := transport.GetTLSConfig(config.CACertificate, config.Certificate, config.PrivateKey)
		if err != nil {
			return err
		}
		transportCreds := credentials.NewTLS(tlsConfig)
		callCreds := credentials.PerRPCCredentials(transport.TokenAuth(config.Token))

		options = append(options,
			grpc.WithTransportCredentials(transportCreds),
			grpc.WithPerRPCCredentials(callCreds),
		)
	}

	h.options = append(h.options, options...)

	return nil
}

func (h *handler) Dial() (rpcClient any, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	host := fmt.Sprintf("%s:%d", h.Config().Host, h.Config().Port)

	conn, err := grpc.DialContext(ctx, host, h.options...)
	if err != nil {
		return nil, err
	}

	h.rpc = proto.NewTeamClient(conn)

	return h.rpc, nil
}

func (h *handler) Close() error {
	return nil
}

// Users returns a list of all users registered to the application server.
func (h *handler) Users() (users []team.User, err error) {
	if h.rpc == nil {
		return nil, errors.New("No working RPC attached to client")
	}

	res, err := h.rpc.GetUsers(context.Background(), &proto.Empty{})
	if err != nil {
		return nil, err
	}

	for _, user := range res.GetUsers() {
		users = append(users, team.User{
			Name:   user.Name,
			Online: user.Online,
		})
	}

	return
}

// ServerVersion returns the version information of the server to which
// the client is connected, or nil and an error if it could not retrieve it.
func (h *handler) Version() (version team.Version, err error) {
	if h.rpc == nil {
		return version, errors.New("No working RPC attached to client")
	}

	ver, err := h.rpc.GetVersion(context.Background(), &proto.Empty{})
	if err != nil {
		return version, errors.New(status.Convert(err).Message())
	}

	return team.Version{
		Major:      ver.Major,
		Minor:      ver.Minor,
		Patch:      ver.Patch,
		Commit:     ver.Commit,
		Dirty:      ver.Dirty,
		CompiledAt: ver.CompiledAt,
		OS:         ver.OS,
		Arch:       ver.Arch,
	}, nil
}

func (h *handler) loggingInterceptor(log *logrus.Entry) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		rawRequest, err := json.Marshal(req)
		if err != nil {
			log.Errorf("Failed to serialize: %w", err)
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		log.Debugf("Raw request: %s", string(rawRequest))

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
