package client

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/reeflective/team"
	"github.com/reeflective/team/internal/log"
)

// Client is a team client wrapper.
// It offers the core functionality of any team client.
type Client struct {
	name         string
	connected    bool
	opts         *opts
	fileLogger   *logrus.Logger
	stdoutLogger *logrus.Logger
	logFile      *os.File
	connect      *sync.Once
	dialer       Dialer[any]
	client       team.Client
}

type Dialer[clientConn any] interface {
	Init(c *Client) error
	Dial() (conn clientConn, err error)
	Close() error
}

func New(application string, teamclient team.Client, options ...Options) (*Client, error) {
	// Client has default logfile path, logging options.
	client := &Client{
		name:    application,
		connect: &sync.Once{},
		client:  teamclient,
	}
	client.opts = client.defaultOpts()

	client.apply(options...)

	// Logging (if allowed)
	if err := client.initLogging(); err != nil {
		return nil, err
	}

	return client, nil
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

	tc.connect.Do(func() {
		_, err = tc.initConfig()
		if err != nil {
			err = tc.logErrorf("%w: %w", ErrConfig, err)
		}

		// Initialize the dialer with our client.
		err = tc.dialer.Init(tc)
		if err != nil {
			err = tc.logErrorf("%w: %w", ErrConfig, err)
			return
		}

		// Connect to the teamserver.
		client, err := tc.dialer.Dial()
		if err != nil {
			err = tc.logErrorf("%w: %w", ErrClient, err)
			return
		}

		// Post-run hooks are used by consumers to further setup/consume
		// the connection after the latter was established. In the case
		// of RPCs, this client is generally used to register them.
		for _, hook := range tc.opts.hooks {
			if err = hook(client); err != nil {
				err = tc.logErrorf("%w: %w", ErrClient, err)
				return
			}
		}
	})

	return
}

// Users returns a list of all users registered to the application server.
func (tc *Client) Users() (users []team.User, err error) {
	if tc.client == nil {
		return nil, ErrNoTeamclient
	}

	res, err := tc.client.Users()
	if err != nil && len(res) == 0 {
		return nil, err
	}

	return res, nil
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
	return tc.log().WithFields(logrus.Fields{
		log.PackageFieldKey: pkg,
		"stream":            stream,
	})
}

// WithLoggerStdout sets the source to which the stdout logger (not any file logger) should write to.
// This option is used by the teamserver/teamclient cobra command tree to coordinate its basic I/O/err.
func (tc *Client) SetLogWriter(stdout, stderr io.Writer) {
	tc.stdoutLogger.Out = stdout
	// TODO: Pass stderr to log internals.
}

// SetLogLevel is a utility to change the logging level of the stdout logger.
func (tc *Client) SetLogLevel(level int) {
	if tc.stdoutLogger == nil {
		return
	}

	if uint32(level) > uint32(logrus.TraceLevel) {
		level = int(logrus.TraceLevel)
	}

	tc.stdoutLogger.SetLevel(logrus.Level(uint32(level)))

	if tc.fileLogger != nil {
		tc.fileLogger.SetLevel(logrus.Level(uint32(level)))
	}
}

// Initialize loggers in files/stdout according to options.
func (tc *Client) initLogging() (err error) {
	// No logging means only stdout with warn level
	if tc.opts.noLogs {
		tc.stdoutLogger = log.NewStdout(tc.Name(), logrus.WarnLevel)
		return nil
	}

	// If user supplied a logger, use it in place of the
	// file-based logger, since the file logger is optional.
	if tc.opts.logger != nil {
		tc.fileLogger = tc.opts.logger
	}

	// Either use default logfile or user-specified one.
	tc.fileLogger, tc.stdoutLogger, err = log.NewClient(tc.opts.logFile, logrus.InfoLevel)
	if err != nil {
		return err
	}

	return nil
}

// log returns a non-nil logger for the client:
// if file logging is disabled, it returns the stdout-only logger,
// otherwise returns the file logger equipped with a stdout hook.
func (tc *Client) log() *logrus.Logger {
	if tc.fileLogger != nil {
		return tc.fileLogger
	}

	if tc.stdoutLogger == nil {
		tc.stdoutLogger = log.NewStdout(tc.Name(), logrus.WarnLevel)
	}

	return tc.stdoutLogger
}

func (tc *Client) logErrorf(msg string, format ...any) error {
	logged := fmt.Errorf(msg, format...)
	tc.log().Errorf(msg, format...)

	return logged
}

func (tc *Client) logError(err error) error {
	tc.log().Error(err)
	return err
}
