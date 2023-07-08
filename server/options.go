package server

import (
	"gorm.io/gorm"

	"github.com/reeflective/team/server/db"
	"github.com/sirupsen/logrus"
)

// Options are server options.
type Options func(opts *opts[any])

type opts[server any] struct {
	logFile     string
	local       bool
	userDefault bool
	noLogs      bool
	noFiles     bool

	config   *Config
	dbConfig *db.Config
	db       *gorm.DB
	logger   *logrus.Logger

	hooks []func(serv server) error
}

// default in-memory configuration, ready to run.
func newDefaultOpts() *opts[any] {
	options := &opts[any]{
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

// WithDefaultPort sets the default port on which the teamserver should start listeners.
// This default is used in the default daemon configuration, and as command flags defaults.
func WithDefaultPort(port uint16) Options {
	return func(opts *opts[any]) {
		opts.config.DaemonMode.Port = int(port)
	}
}

// WithNoFiles deactivates all interactions between the teamserver and
// the OS filesystem: no database is created, no log files written.
// Using this option with noFiles set to true will in effect disable
// the multiplayer/remote functionality of the teamserver.
//
// This option can be useful if you have embedded a teamserver into
// your application because you might need it in the future, but that
// you don't want it yet to do anything other than being compiled in.
func WithNoFiles(noFiles bool) Options {
	return func(opts *opts[any]) {
		opts.noFiles = noFiles
	}
}

// WithNoLogs deactivates all logging normally done by the teamserver
// if noLogs is set to true, or keeps/reestablishes them if false.
func WithNoLogs(noLogs bool) Options {
	return func(opts *opts[any]) {
		opts.noLogs = noLogs
	}
}

// WithLogFile sets the path to the file where teamserver logging should be done.
func WithLogFile(filePath string) Options {
	return func(opts *opts[any]) {
		opts.logFile = filePath
	}
}

// WithLogger sets the teamserver to use a specific logger for
// all logging, except the audit log which is indenpendent.
func WithLogger(logger *logrus.Logger) Options {
	return func(opts *opts[any]) {
		opts.logger = logger
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
// func WithOSUserDefault() Options {
// 	return func(opts *opts[any]) {
// 		opts.userDefault = true
// 	}
// }
