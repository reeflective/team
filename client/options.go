package client

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"io"

	"github.com/sirupsen/logrus"
)

// Options are client options.
type Options func(opts *opts)

type opts struct {
	noLogs   bool
	logFile  string
	inMemory bool
	console  bool
	local    bool
	stdout   io.Writer
	config   *Config
	logger   *logrus.Logger
	dialer   Dialer[any]
	hooks    []func(s any) error
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

	if tc.opts.stdout != nil {
		tc.stdoutLogger.Out = tc.opts.stdout
	}
}

//
// *** General options ***
//

// WithInMemory deactivates all interactions of the client with the filesystem.
// This applies to logging, but will also to any forward feature using files.
func WithInMemory() Options {
	return func(opts *opts) {
		opts.noLogs = true
		opts.inMemory = true
	}
}

// WithConfig sets the client to use a given teamserver configuration for
// connection, instead of using default user/application configurations.
func WithConfig(config *Config) Options {
	return func(opts *opts) {
		opts.config = config
	}
}

//
// *** Logging options ***
//

// WithNoLogs deactivates all logging normally done by the teamclient
// if noLogs is set to true, or keeps/reestablishes them if false.
func WithNoLogs(noLogs bool) Options {
	return func(opts *opts) {
		opts.noLogs = noLogs
	}
}

// WithLogFile sets the path to the file where teamclient logging should be done.
// If not specified, the client log file is ~/.app/teamclient/logs/app.teamclient.log.
func WithLogFile(filePath string) Options {
	return func(opts *opts) {
		opts.logFile = filePath
	}
}

// WithLogger sets the teamclient to use a specific logger for logging.
func WithLogger(logger *logrus.Logger) Options {
	return func(opts *opts) {
		opts.logger = logger
	}
}

//
// *** Client network/RPC options ***
//

// WithDialer sets a custom dialer to connect to the teamserver.
func WithDialer(dialer Dialer[any]) Options {
	return func(opts *opts) {
		opts.dialer = dialer
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

// WithPostConnectHooks adds a list of hooks to run on the generic RPC client
// returned by the Teamclient/Dialer Dial() method. This client object can be
// pretty much any client-side RPC connection, or just raw connection.
// You will have to typecast this conn in your hooks.
func WithPostConnectHooks(hooks ...func(conn any) error) Options {
	return func(opts *opts) {
		opts.hooks = append(opts.hooks, hooks...)
	}
}
