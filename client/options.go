package client

import "google.golang.org/grpc"

// Options are client options.
type Options func(opts *opts) *opts

type opts struct {
	config *Config
	conn   *grpc.ClientConn
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
