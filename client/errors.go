package client

import "errors"

var (
	// ErrNoTeamclient indicates that the client cannot remotely query a server
	// to get its version or user information, because there is no client RPC
	// to do it. Make sure that your team/client.Client has been given one.
	ErrNoTeamclient = errors.New("this teamclient has no client implementation")

	// ErrConfig is an error related to the teamclient connection configuration.
	ErrConfig = errors.New("client config error")

	// ErrConfigNoUser says that the configuration has no user,
	// which is not possible even if the client is an in-memory one.
	ErrConfigNoUser = errors.New("client config with empty user")
)
