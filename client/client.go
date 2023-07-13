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
	"io/fs"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/reeflective/team"
	"github.com/reeflective/team/internal/assets"
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

	// Filesystem (in-memory or on disk)
	user, _ := user.Current()
	root := filepath.Join(user.HomeDir, "."+client.name)
	client.fs = assets.NewFileSystem(root, client.opts.inMemory)

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
