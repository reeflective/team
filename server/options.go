package server

import (
	"gorm.io/gorm"

	"github.com/reeflective/team/server/db"
)

// Options are server options.
type Options func(opts *opts) *opts

type opts struct {
	db            *gorm.DB
	dbConfig      *db.Config
	preServeHooks []func(s *Server) error
}

func (s *Server) apply(options ...Options) {
	for _, optFunc := range options {
		optFunc(s.opts)
	}
}

// WithDatabaseConfig sets the server to use a database backend with a given configuration.
func WithDatabaseConfig(config *db.Config) Options {
	return func(opts *opts) *opts {
		opts.dbConfig = config
		return opts
	}
}

// WithDatabase sets the server database to an existing database.
// Note that it will run an automigration of the teamserver types (certificates and users).
func WithDatabase(db *gorm.DB) Options {
	return func(opts *opts) *opts {
		opts.db = db
		return opts
	}
}

// WithPreServeHooks is used to register additional steps to the teamserver "before" serving
// its gRPC server connection and services. While this is not needed when your code path allows
// you to further manipulate the server connection after start, it is useful for persistent jobs
// that restarted on server start: in order to bind your application functionality to them, you
// need to use register hooks here.
func WithPreServeHooks(hooks ...func(s *Server) error) Options {
	return func(opts *opts) *opts {
		opts.preServeHooks = append(opts.preServeHooks, hooks...)
		return opts
	}
}
