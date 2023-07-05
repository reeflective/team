package client

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/reeflective/team/internal/proto"
	"github.com/sirupsen/logrus"
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
	log        *slog.Logger
	logger     *logrus.Logger
	logFile    *os.File
	conn       *grpc.ClientConn
	rpc        proto.TeamClient
	connectedT *sync.Once
}

// Name returns the name of the client application.
func (c *Client) Name() string {
	return c.name
}

// New returns an application client ready to work.
// The application client log file is opened and served to the client builtin logger.
// The client will panic if it can't open or create this log file as ~/.app/client.log.
func New(application string, options ...Options) *Client {
	c := &Client{
		opts:       &opts{},
		name:       application,
		connected:  false,
		connectedT: &sync.Once{},
	}

	// c.logFile = c.initLogging(c.AppDir())

	c.apply(options...)

	return c
}

// Connect uses the default client configurations to connect to the team server.
// Note that this call might be blocking and expect user input, in the case where more
// than one server configuration is found in the application directory: the application
// will prompt the user to choose one of them.
func (c *Client) Connect(options ...Options) (err error) {
	// There is no way to return an error from client RPC registration.
	// Everything will be found in the logs, but in the meantime, let's
	// consider us connected, until we explicitely Disconnect().
	defer c.connectedT.Do(func() {
		c.rpc = proto.NewTeamClient(c.conn)
	})

	c.apply(options...)

	// A non-nil connection means we already have a precise target:
	// currently we don't use the experimental grpc.Connect() method,
	// which should mean reuse, so once we were disconnected from any
	// previous stream -over memory or TLS, regardless- we dropped it.
	if c.conn != nil {
		return
	}

	var cfg *Config

	// Else connect with any available configuration.
	if c.opts.config != nil {
		cfg = c.opts.config
	} else {
		configs := c.GetConfigs()
		if len(configs) == 0 {
			return fmt.Errorf("no config files found at %s", c.ConfigsDir())
		}
		cfg = c.SelectConfig()
	}

	if cfg == nil {
		return errors.New("no application was selected or parsed")
	}

	// Establish the connection and bind RPC core.
	c.conn, err = c.connect(cfg)

	if err == nil && c.conn != nil {
		c.connected = true
	}

	return
}

// Disconnect disconnects the client from the server, closing the connection
// and the client log file.Any errors are logged to the this file, not returned.
func (c *Client) Disconnect() {
	if c.opts.console {
		return
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.log.Error(fmt.Sprintf("error closing connection: %v", err))
		}
	}

	if c.logFile != nil {
		c.logFile.Close()
	}

	// Decrement the counter, should be back to 0.
	c.connected = false
	c.conn = nil
	c.connectedT = &sync.Once{}
}

// Connection returns the gRPC client connection it uses.
func (c *Client) Connection() *grpc.ClientConn {
	return c.conn
}

// IsConnected returns true if a working teamclient to server connection
// is bound to to this precise client. Given that each client register may
// register as many other RPC client services to its connection, this client
// can't however reconnect to/with a different connection/stream.
func (c *Client) IsConnected() bool {
	return c.connected
}

// Users returns a list of all users registered to the application server.
func (c *Client) Users() (users []proto.User) {
	if c.rpc == nil {
		return nil
	}

	res, err := c.rpc.GetUsers(context.Background(), &proto.Empty{})
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
func (c *Client) ServerVersion() (ver *proto.Version, err error) {
	if c.rpc == nil {
		return nil, errors.New("no working client RPC is attached to this client")
	}

	res, err := c.rpc.GetVersion(context.Background(), &proto.Empty{})
	if err != nil {
		return nil, errors.New(status.Convert(err).Message())
	}

	return res, nil
}
