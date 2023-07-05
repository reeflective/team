package client

import "google.golang.org/grpc"

// Options are client options.
type Options func(opts *opts) *opts

type opts struct {
	config  *Config
	conn    *grpc.ClientConn
	console bool
}

func (c *Client) apply(options ...Options) {
	for _, optFunc := range options {
		c.opts = optFunc(c.opts)
	}

	if c.opts.conn != nil {
		c.conn = c.opts.conn
	}
}

// WithConnection sets up the client to use a given gRPC client connection.
// This should generally only be used by server binary to speak to themselves.
func WithConnection(conn *grpc.ClientConn) Options {
	return func(opts *opts) *opts {
		opts.conn = conn
		return opts
	}
}

// WithConfig sets the client to use a given teamserver configuration for
// connection, instead of using default user/application configurations.
func WithConfig(config *Config) Options {
	return func(opts *opts) *opts {
		opts.config = config
		return opts
	}
}

// WithNoDisconnect is meant to be used when the teamclient commands are used
// in your application and that you happen to ALSO have a readline/console style
// application which might reuse commands.
// If this is the case, this option will ensure that any cobra client command
// runners produced by this library will not disconnect after each execution.
func WithNoDisconnect() Options {
	return func(opts *opts) *opts {
		opts.console = true
		return opts
	}
}
