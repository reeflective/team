package client

import (
	"fmt"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// Options are client options.
type Options func(opts *opts)

type opts struct {
	noLogs  bool
	logFile string
	console bool
	local   bool
	config  *Config
	logger  *logrus.Logger
	dialer  Dialer[any]
	hooks   []func(s any) error
}

func (tc *Client) defaultOpts() *opts {
	return &opts{
		config:  &Config{},
		logFile: filepath.Join(tc.LogsDir(), fmt.Sprintf("%s.teamclient.log", tc.Name())),
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

// WithInMemory deactivates all interactions of the client with the filesystem.
// This applies to logging, but will also to any forward feature using files.
func WithInMemory() Options {
	return func(opts *opts) {
		opts.noLogs = true
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
// If not specified, the client log file is ~/.app/teamclient/logs/app.teamclient.log
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

// WithLocalDialer sets the teamclient to connect with an in-memory dialer
// (provided when creating the teamclient). This in effect only prevents
// the teamclient from looking and loading/prompting remote client configs.
//
// Because this option is automatically called by the teamserver.ServeLocal()
// function, you should probably not have any reason to use this option.
func WithLocalDialer() Options {
	return func(opts *opts) {
		opts.local = true
	}
}
