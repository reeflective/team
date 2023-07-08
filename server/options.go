package server

import (
	"gorm.io/gorm"

	"github.com/reeflective/team/server/db"
)

// Options are server options.
type Options func(opts *opts[any])

type opts[server any] struct {
	config      *Config
	dbConfig    *db.Config
	db          *gorm.DB
	local       bool
	userDefault bool
	noLogs      bool

	handler func(ln Handler[any]) error
	hooks   []func(serv server) error
}

// default in-memory configuration, ready to run.
func newDefaultOpts[server any]() *opts[server] {
	options := &opts[server]{
		config: getDefaultServerConfig(),
		local:  false,
	}

	return options
}

func (ts *Server) apply(options ...Options) {
	for _, optFunc := range options {
		optFunc(ts.opts)
	}
}

// WithLogger
// WithAuditFile
// WithLogFile

func WithNoLogs() Options {
	return func(opts *opts[any]) {
		opts.noLogs = true
	}
}

// WithDefaultPort sets the default port on which the teamserver should start listeners.
// This default is used in the default daemon configuration, and as command flags defaults.
func WithDefaultPort(port uint16) Options {
	return func(opts *opts[any]) {
		opts.config.DaemonMode.Port = int(port)
	}
}

// WithDatabaseConfig sets the server to use a database backend with a given configuration.
func WithDatabaseConfig(config *db.Config) Options {
	return func(opts *opts[any]) {
		opts.dbConfig = config
	}
}

// WithDatabase sets the server database to an existing database.
// Note that it will run an automigration of the teamserver types (certificates and users).
func WithDatabase(db *gorm.DB) Options {
	return func(opts *opts[any]) {
		opts.db = db
	}
}

func WithCustomHandler(handler func(ln Handler[any]) error) Options {
	return func(opts *opts[any]) {
		opts.handler = handler
	}
}

// WithPreServeHooks is used to register additional steps to the teamserver "before" serving
// its gRPC server connection and services. While this is not needed when your code path allows
// you to further manipulate the server connection after start, it is useful for persistent jobs
// that restarted on server start: in order to bind your application functionality to them, you
// need to use register hooks here.
func WithPreServeHooks(hooks ...func(server any) error) Options {
	return func(opts *opts[any]) {
		opts.hooks = append(opts.hooks, hooks...)
	}
}

// WithOSUserDefault automatically creates a user for the teamserver, using the current OS user.
// This will create the client application directory (~/.app) if needed, and will write the config
// in the configs dir, using 'app_local_user_default.cfg' name, overwriting any file having this name.
func WithOSUserDefault() Options {
	return func(opts *opts[any]) {
		opts.userDefault = true
	}
}
