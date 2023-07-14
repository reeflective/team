package server

import (
	"github.com/reeflective/team/internal/db"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Options are server options.
type Options func(opts *opts[any])

type opts[server any] struct {
	logFile         string
	local           bool
	noLogs          bool
	inMemory        bool
	continueOnError bool

	config    *Config
	dbConfig  *db.Config
	db        *gorm.DB
	logger    *logrus.Logger
	listeners []Handler[server]

	hooks map[string][]func(serv server) error
}

// default in-memory configuration, ready to run.
func newDefaultOpts() *opts[any] {
	options := &opts[any]{
		config: getDefaultServerConfig(),
		hooks:  map[string][]func(serv any) error{},
		local:  false,
	}

	return options
}

func (ts *Server) apply(options ...Options) {
	for _, optFunc := range options {
		optFunc(ts.opts)
	}

	if ts.opts.db != nil {
		ts.db = ts.opts.db
	}

	// Load any listener backends.
	for _, listener := range ts.opts.listeners {
		ts.handlers[listener.Name()] = listener
	}

	// Make the first one as the default if needed.
	if len(ts.opts.listeners) > 0 && ts.self == nil {
		ts.self = ts.opts.listeners[0]
	}

	ts.opts.listeners = make([]Handler[any], 0)
}

//
// *** General options ***
//

// WithInMemory deactivates all interactions of the client with the filesystem.
// This applies to logging, but will also to any forward feature using files.
//
// Implications on database backends:
// By default, all teamservers use sqlite3 as a backend, and thus will run a
// database in memory. All other databases are assumed to be unable to do so,
// and this option will thus trigger an error whenever the option is applied,
// whether it be at teamserver creation, or when it does start listeners.
func WithInMemory() Options {
	return func(opts *opts[any]) {
		opts.noLogs = true
		opts.inMemory = true
	}
}

// WithDefaultPort sets the default port on which the teamserver should start listeners.
// This default is used in the default daemon configuration, and as command flags defaults.
// The default port set for teamserver applications is port 31416.
func WithDefaultPort(port uint16) Options {
	return func(opts *opts[any]) {
		opts.config.DaemonMode.Port = int(port)
	}
}

// WithDatabase sets the server database to an existing database.
// Note that it will run an automigration of the teamserver types (certificates and users).
func WithDatabase(db *gorm.DB) Options {
	return func(opts *opts[any]) {
		opts.db = db
	}
}

// WithDatabaseConfig sets the server to use a database backend with a given configuration.
func WithDatabaseConfig(config *db.Config) Options {
	return func(opts *opts[any]) {
		opts.dbConfig = config
	}
}

//
// *** Logging options ***
//

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

//
// *** Server network/RPC options ***
//

func WithListener(ln Handler[any]) Options {
	return func(opts *opts[any]) {
		opts.listeners = append(opts.listeners, ln)
	}
}

// WithContinueOnError sets the teamserver behavior when starting persistent listeners
// (either automatically when calling teamserver.ServeDaemon(), or when using
// teamserver.StartPersistentListeners()).
// If true, an error raised by a listener will not prevent others to try starting, and
// errors will be joined into a single one, separated with newlines and logged by default.
// The teamserver has this set to false by default.
func WithContinueOnError(continueOnError bool) Options {
	return func(opts *opts[any]) {
		opts.continueOnError = continueOnError
	}
}

// WithPreServeHooks is used to register additional steps to the teamserver "before" serving
// its gRPC server connection and services. While this is not needed when your code path allows
// you to further manipulate the server connection after start, it is useful for persistent jobs
// that restarted on server start: in order to bind your application functionality to them, you
// need to use register hooks here.
func WithPreServeHooks(handlerName string, hooks ...func(server any) error) Options {
	return func(opts *opts[any]) {
		opts.hooks[handlerName] = append(opts.hooks[handlerName], hooks...)
	}
}
