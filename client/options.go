package client

import "github.com/sirupsen/logrus"

// Options are client options.
type Options func(opts *opts)

type opts struct {
	config  *Config
	noLogs  bool
	logFile string
	logger  *logrus.Logger
	console bool
	dialer  Dialer[any]
	hooks   []func(s any) error
}

func defaultOpts() *opts {
	return &opts{
		config: &Config{},
	}
}

func (tc *Client) apply(options ...Options) {
	for _, optFunc := range options {
		optFunc(tc.opts)
	}

	if tc.opts.dialer != nil {
		tc.dialer = tc.opts.dialer
	}
}

// WithNoLogs deactivates all logging normally done by the teamclient
// if noLogs is set to true, or keeps/reestablishes them if false.
func WithNoLogs(noLogs bool) Options {
	return func(opts *opts) {
		opts.noLogs = noLogs
	}
}

// WithLogFile sets the path to the file where teamclient logging should be done.
func WithLogFile(filePath string) Options {
	return func(opts *opts) {
		opts.logFile = filePath
	}
}

// WithLogger sets the teamclient to use a specific logger for logging
func WithLogger(logger *logrus.Logger) Options {
	return func(opts *opts) {
		opts.logger = logger
	}
}

// WithConfig sets the client to use a given teamserver configuration for
// connection, instead of using default user/application configurations.
func WithConfig(config *Config) Options {
	return func(opts *opts) {
		opts.config = config
	}
}

// WithNoDisconnect is meant to be used when the teamclient commands are used
// in your application and that you happen to ALSO have a readline/console style
// application which might reuse commands.
// If this is the case, this option will ensure that any cobra client command
// runners produced by this library will not disconnect after each execution.
func WithNoDisconnect() Options {
	return func(opts *opts) {
		opts.console = true
	}
}

// WithDialer sets a custom dialer to connect to the teamserver.
func WithDialer(dialer Dialer[any]) Options {
	return func(opts *opts) {
		opts.dialer = dialer
	}
}

// WithPostConnectHooks adds a list of hooks to run on the generic RPC client
// returned by the Teamclient/Dialer Dial() method. This client object can be
// pretty much any client-side RPC connection, or just raw connection.
// You will have to typecast this conn in your hooks.
func WithPostConnectHooks(hooks ...func(conn any) error) Options {
	return func(opts *opts) {
		opts.hooks = append(opts.hooks, hooks...)
	}
}
