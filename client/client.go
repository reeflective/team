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
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/reeflective/team"
	"github.com/reeflective/team/internal/assets"
	"github.com/reeflective/team/internal/log"
	"github.com/sirupsen/logrus"
)

// Client is a team client wrapper.
// It offers the core functionality of any team client.
type Client struct {
	name         string
	opts         *opts
	fileLogger   *logrus.Logger
	stdoutLogger *logrus.Logger
	logFile      fs.File
	fs           *assets.FS

	dialer  Dialer[any]
	connect *sync.Once

	mutex  *sync.RWMutex
	client team.Client
}

type Dialer[clientConn any] interface {
	Init(c *Client) error
	Dial() (conn clientConn, err error)
	Close() error
}

func New(application string, teamclient team.Client, options ...Options) (*Client, error) {
	client := &Client{
		name:    application,
		opts:    defaultOpts(),
		client:  teamclient,
		connect: &sync.Once{},
		mutex:   &sync.RWMutex{},
		fs:      &assets.FS{},
	}

	client.apply(options...)

	// Filesystem
	client.fs = assets.NewFileSystem(client.opts.inMemory)

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
		return nil
	}

	tc.connect.Do(func() {
		err = tc.initConfig()
		if err != nil {
			err = tc.errorf("%w: %w", ErrConfig, err)
			return
		}

		// Initialize the dialer with our client.
		err = tc.dialer.Init(tc)
		if err != nil {
			err = tc.errorf("%w: %w", ErrConfig, err)
			return
		}

		// Connect to the teamserver.
		var client any

		client, err = tc.dialer.Dial()
		if err != nil {
			err = tc.errorf("%w: %w", ErrClient, err)
			return
		}

		// Post-run hooks are used by consumers to further setup/consume
		// the connection after the latter was established. In the case
		// of RPCs, this client is generally used to register them.
		for _, hook := range tc.opts.hooks {
			if err = hook(client); err != nil {
				err = tc.errorf("%w: %w", ErrClient, err)
				return
			}
		}
	})

	return err
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
func (tc *Client) Disconnect() error {
	if tc.opts.console {
		return nil
	}

	// The client can reconnect..
	defer func() {
		tc.connect = &sync.Once{}
	}()

	if tc.dialer == nil {
		return nil
	}

	err := tc.dialer.Close()
	if err != nil {
		tc.log().Error(err)
	}

	return err
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

	// if tc.fileLogger != nil {
	// 	tc.fileLogger.SetLevel(logrus.Level(uint32(level)))
	// }
}

// Initialize loggers in files/stdout according to options.
func (tc *Client) initLogging() (err error) {
	// By default, the stdout logger is never nil.
	// We might overwrite it below if using our defaults.
	tc.stdoutLogger = log.NewStdio(logrus.WarnLevel)

	// Path to our client log file, and open it (in mem or on disk)
	logFile := filepath.Join(tc.LogsDir(), log.FileName(tc.Name(), false))

	// If the teamclient should log to a predefined file.
	if tc.opts.logFile != "" {
		logFile = tc.opts.logFile
	}

	// If user supplied a logger, use it in place of the
	// file-based logger, since the file logger is optional.
	if tc.opts.logger != nil {
		tc.fileLogger = tc.opts.logger
		return nil
	}

	// Create the loggers writing to this file, and hooked to write to stdout as well.
	tc.fileLogger, tc.stdoutLogger, err = log.Init(tc.fs, logFile, logrus.InfoLevel)
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
		tc.stdoutLogger = log.NewStdio(logrus.WarnLevel)
	}

	return tc.stdoutLogger
}

func (tc *Client) errorf(msg string, format ...any) error {
	logged := fmt.Errorf(msg, format...)
	tc.log().Error(logged)

	return logged
}
