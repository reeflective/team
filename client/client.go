package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/internal/proto"
)

const (
	// configsDirName - Directory name containing config files
	configsDirName  = "configs"
	versionFileName = "version"
	logFileName     = "client.log"
)

// Client is a team client wrapper.
// It offers the core functionality of any team client.
type Client struct {
	name       string
	connected  bool
	opts       *opts
	log        *logrus.Logger
	logFile    *os.File
	conn       *grpc.ClientConn
	rpc        proto.TeamClient
	connectedT *sync.Once
}

// Name returns the name of the client application.
func (tc *Client) Name() string {
	return tc.name
}

// New returns an application client ready to work.
// The application client log file is opened and served to the client builtin logger.
// The client will panic if it can't open or create this log file as ~/.app/client.log.
func New(application string, options ...Options) (*Client, error) {
	var err error

	client := &Client{
		opts:       &opts{},
		name:       application,
		connected:  false,
		connectedT: &sync.Once{},
	}

	client.apply(options...)

	// Loggers
	client.log, err = log.NewClient(client.AppDir(), application, logrus.ErrorLevel)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Connect uses the default client configurations to connect to the team server.
// Note that this call might be blocking and expect user input, in the case where more
// than one server configuration is found in the application directory: the application
// will prompt the user to choose one of them.
func (tc *Client) Connect(options ...Options) (err error) {
	// There is no way to return an error from client RPC registration.
	// Everything will be found in the logs, but in the meantime, let's
	// consider us connected, until we explicitely Disconnect().
	defer tc.connectedT.Do(func() {
		tc.rpc = proto.NewTeamClient(tc.conn)
	})

	tc.apply(options...)

	// A non-nil connection means we already have a precise target:
	// currently we don't use the experimental grpc.Connect() method,
	// which should mean reuse, so once we were disconnected from any
	// previous stream -over memory or TLS, regardless- we dropped it.
	if tc.conn != nil {
		return
	}

	var cfg *Config

	// Else connect with any available configuration.
	if tc.opts.config != nil {
		cfg = tc.opts.config
	} else {
		configs := tc.GetConfigs()
		if len(configs) == 0 {
			return fmt.Errorf("no config files found at %s", tc.ConfigsDir())
		}
		cfg = tc.SelectConfig()
	}

	if cfg == nil {
		return errors.New("no application was selected or parsed")
	}

	// Establish the connection and bind RPC core.
	tc.conn, err = tc.connect(cfg)

	if err == nil && tc.conn != nil {
		tc.connected = true
	}

	return
}

// Disconnect disconnects the client from the server, closing the connection
// and the client log file.Any errors are logged to the this file, not returned.
func (tc *Client) Disconnect() {
	if tc.opts.console {
		return
	}

	if tc.conn != nil {
		if err := tc.conn.Close(); err != nil {
			tc.log.Error(fmt.Sprintf("error closing connection: %v", err))
		}
	}

	if tc.logFile != nil {
		tc.logFile.Close()
	}

	// Decrement the counter, should be back to 0.
	tc.connected = false
	tc.conn = nil
	tc.connectedT = &sync.Once{}
}

// Connection returns the gRPC client connection it uses.
func (tc *Client) Connection() *grpc.ClientConn {
	return tc.conn
}

// IsConnected returns true if a working teamclient to server connection
// is bound to to this precise client. Given that each client register may
// register as many other RPC client services to its connection, this client
// can't however reconnect to/with a different connection/stream.
func (tc *Client) IsConnected() bool {
	return tc.connected
}

// Users returns a list of all users registered to the application server.
func (tc *Client) Users() (users []proto.User) {
	if tc.rpc == nil {
		return nil
	}

	res, err := tc.rpc.GetUsers(context.Background(), &proto.Empty{})
	if err != nil {
		return nil
	}

	for _, user := range res.GetUsers() {
		users = append(users, *user)
	}

	return
}

// ServerVersion returns the version information of the server to which
// the client is connected, or nil and an error if it could not retrieve it.
func (tc *Client) ServerVersion() (ver *proto.Version, err error) {
	if tc.rpc == nil {
		return nil, errors.New("no working client RPC is attached to this client")
	}

	res, err := tc.rpc.GetVersion(context.Background(), &proto.Empty{})
	if err != nil {
		return nil, errors.New(status.Convert(err).Message())
	}

	return res, nil
}
