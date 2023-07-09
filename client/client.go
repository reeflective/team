package client

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/reeflective/team"
	"github.com/reeflective/team/internal/log"
)

var ErrNoTeamclient = errors.New("This teamclient has no client implementation")

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
	client     team.Client
}

type Dialer[clientConn any] interface {
	Init(c *Client) error
	Dial() (conn clientConn, err error)
	Close() error
}

func New(application string, client team.Client, options ...Options) (*Client, error) {
	var err error

	teamclient := &Client{
		name:       application,
		opts:       defaultOpts(),
		connectedT: &sync.Once{},
		client:     client,
	}

	teamclient.apply(options...)

	// Loggers
	teamclient.log, err = log.NewClient(teamclient.AppDir(), application, logrus.DebugLevel)
	if err != nil {
		return nil, err
	}

	return teamclient, nil
}

// Connect uses the default client configurations to connect to the team server.
// Note that this call might be blocking and expect user input, in the case where more
// than one server configuration is found in the application directory: the application
// will prompt the user to choose one of them.
//
// It only connects the teamclient if it has an available dialer.
// If none is available, this function returns no error, as it is
// possible that this client has a teamclient implementation ready.
func (tc *Client) Connect(options ...Options) (err error) {
	tc.apply(options...)

	if tc.dialer == nil {
		return
	}

	tc.connectedT.Do(func() {
		cfg := tc.opts.config

		if !tc.opts.local {
			configs := tc.GetConfigs()
			if len(configs) == 0 {
				err = fmt.Errorf("no config files found at %s", tc.ConfigsDir())
				return
			}
			cfg = tc.SelectConfig()
		}

		if cfg == nil {
			err = errors.New("no application was selected or parsed")
		}
		tc.opts.config = cfg

		// Initialize the dialer with our client.
		err = tc.dialer.Init(tc)
		if err != nil {
			return
		}

		// Connect to the teamserver.
		client, err := tc.dialer.Dial()
		if err != nil {
			return
		}

		// Post-run hooks are used by consumers to further setup/consume
		// the connection after the latter was established. In the case
		// of RPCs, this client is generally used to register them.
		for _, hook := range tc.opts.hooks {
			if err = hook(client); err != nil {
				return
			}
		}
	})

	return
}

// Users returns a list of all users registered to the application server.
func (tc *Client) Users() (users []team.User) {
	if tc.client == nil {
		return nil
	}

	res, err := tc.client.Users()
	if err != nil && len(res) == 0 {
		return nil
	}

	return res
}

// ServerVersion returns the version information of the server to which
// the client is connected, or nil and an error if it could not retrieve it.
func (tc *Client) ServerVersion() (ver team.Version, err error) {
	if tc.client == nil {
		return ver, ErrNoTeamclient
	}

	version, err := tc.client.Version()
	if err != nil {
		return
	}

	return version, nil
}

// Name returns the name of the client application.
func (tc *Client) Name() string {
	return tc.name
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

// NamedLogger returns a new logging "thread" which should grossly
// indicate the package/general domain, and a more precise flow/stream.
func (tc *Client) NamedLogger(pkg, stream string) *logrus.Entry {
	return tc.log.WithFields(logrus.Fields{
		log.PackageFieldKey: pkg,
		"stream":            stream,
	})
}
