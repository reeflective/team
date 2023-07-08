package client

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/internal/proto"
)

// Client is a team client wrapper.
// It offers the core functionality of any team client.
type Client struct {
	name       string
	connected  bool
	opts       *opts
	log        *logrus.Logger
	logFile    *os.File
	connectedT *sync.Once
	dialer     Dialer[any]
	core       Teamclient[any]
}

type Teamclient[rpcClient any] interface {
	Dialer[rpcClient]

	Users() ([]*proto.User, error)
	Version() (*proto.Version, error)
}

type Dialer[rpcClient any] interface {
	Init(c *Client) error
	Dial() (conn rpcClient, err error)
	Close() error
}

func New(application string, client Teamclient[any], options ...Options) (*Client, error) {
	var err error

	teamclient := &Client{
		name:       application,
		opts:       defaultOpts(),
		core:       client,
		connectedT: &sync.Once{},
	}

	// Loggers
	teamclient.log, err = log.NewClient(teamclient.AppDir(), application, logrus.ErrorLevel)
	if err != nil {
		return nil, err
	}

	return teamclient, nil
}

// Name returns the name of the client application.
func (tc *Client) Name() string {
	return tc.name
}

// Connect uses the default client configurations to connect to the team server.
// Note that this call might be blocking and expect user input, in the case where more
// than one server configuration is found in the application directory: the application
// will prompt the user to choose one of them.
func (tc *Client) Connect(options ...Options) (err error) {
	tc.apply(options...)

	cfg := tc.opts.config

	// Else connect with any available configuration.
	// TODO: Change this, since config is never nil...
	if tc.opts.config != nil {
		configs := tc.GetConfigs()
		if len(configs) == 0 {
			return fmt.Errorf("no config files found at %s", tc.ConfigsDir())
		}
		cfg = tc.SelectConfig()
	}

	if cfg == nil {
		return errors.New("no application was selected or parsed")
	}
	tc.opts.config = cfg

	// Our teamclient must be a dialer, only use
	// it if we don't have a custom dialer to use
	if tc.dialer == nil {
		tc.dialer = tc.core
	}

	// Initialize the dialer with our client.
	err = tc.dialer.Init(tc)
	if err != nil {
		return err
	}

	// Connect to the teamserver.
	client, err := tc.dialer.Dial()
	if err != nil {
		return err
	}

	// Post-run hooks are used by consumers to further setup/consume
	// the connection after the latter was established. In the case
	// of RPCs, this client is generally used to register them.
	for _, hook := range tc.opts.hooks {
		if err := hook(client); err != nil {
			return err
		}
	}

	return
}

// Disconnect disconnects the client from the server, closing the connection
// and the client log file.Any errors are logged to the this file, not returned.
func (tc *Client) Disconnect() {
	if tc.opts.console {
		return
	}

	// if tc.conn != nil {
	// 	if err := tc.conn.Close(); err != nil {
	// 		tc.log.Error(fmt.Sprintf("error closing connection: %v", err))
	// 	}
	// }
	//
	if tc.logFile != nil {
		tc.logFile.Close()
	}
	//
	// // Decrement the counter, should be back to 0.
	// tc.connected = false
	// tc.conn = nil
	// tc.connectedT = &sync.Once{}
	return
}

// IsConnected returns true if a working teamclient to server connection
// is bound to to this precise client. Given that each client register may
// register as many other RPC client services to its connection, this client
// can't however reconnect to/with a different connection/stream.
func (tc *Client) IsConnected() bool {
	return tc.connected
}

// Users returns a list of all users registered to the application server.
func (tc *Client) Users() (users []*proto.User) {
	if tc.core == nil {
		return nil
	}

	res, err := tc.core.Users()
	if err != nil {
		return nil
	}

	return res
}

// ServerVersion returns the version information of the server to which
// the client is connected, or nil and an error if it could not retrieve it.
func (tc *Client) ServerVersion() (ver proto.Version, err error) {
	if tc.core == nil {
		return ver, errors.New("No working RPC")
	}

	version, err := tc.core.Version()
	if err != nil || version == nil {
		return ver, err
	}

	return *version, nil
}

// NamedLogger returns a new logging "thread" which should grossly
// indicate the package/general domain, and a more precise flow/stream.
func (tc *Client) NamedLogger(pkg, stream string) *logrus.Entry {
	return log.NewNamed(tc.log, pkg, stream)
}
