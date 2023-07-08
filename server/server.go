package server

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/certs"
	"github.com/reeflective/team/internal/log"
	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/server/db"
)

// Server is a team server.
type Server struct {
	// Core
	name       string
	rootDirEnv string
	listening  bool
	log        *logrus.Logger
	userTokens *sync.Map

	// Configurations
	opts  *opts[any]
	db    *gorm.DB
	certs *certs.Manager

	ln Handler[any]

	// Services
	initOnce *sync.Once
	*proto.UnimplementedTeamServer
}

// Handler represents a teamserver listener stack.
// It must be satisfied by any type aiming to be used
// as a transport and RPC backend of the teamserver.
//
// The server type parameter represents the server-side of a connection
// between a teamclient and the teamserver. It may or may not offer the
// RPC services to the client yet. Implementations are free to decide.
// TODO: Write about the Options(hooks) to use with this generic type.
type Handler[server any] interface {
	Init(s *Server) error
	Listen(addr string) (ln net.Listener, err error)
	Serve(net.Listener) (serv server, err error)
	Close() error
}

// New creates a new teamserver for the provided application name.
// This server can handle any number of remote clients for a given application
// named "teamserver", including its own local runtime (fully in-memory) client.
//
// All errors returned from this call are critical, in that the server could not
// run properly in its most basic state. If an error is raised, no server is returned.
//
// This call to create the server only creates the application default directory.
// No files, logs, connections or any interaction with the os/filesystem are made.
func New(application string, ln Handler[any], options ...Options) (*Server, error) {
	var err error

	server := &Server{
		name:                    application,
		rootDirEnv:              fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(application)),
		userTokens:              &sync.Map{},
		opts:                    newDefaultOpts(),
		ln:                      ln,
		initOnce:                &sync.Once{},
		UnimplementedTeamServer: &proto.UnimplementedTeamServer{},
	}

	server.apply(options...)

	// Logging
	level := logrus.Level(server.opts.config.Log.Level)

	// Ensure all teamserver-specific directories are writable.
	if !server.opts.noLogs || !server.opts.noFiles {
		if err := server.checkWritableFiles(); err != nil {
			return nil, err
		}
		server.log, err = log.NewClient(server.LogsDir(), server.Name(), level)
		if err != nil {
			return nil, err
		}
	} else {
		server.log = log.NewStdout(server.Name(), level)
	}

	return server, nil
}

// Name returns the name of the application handled by the teamserver.
// Since you can embed multiple teamservers (one for each application)
// into a single binary, this is different from the program binary name
// running this teamserver.
func (ts *Server) Name() string {
	return ts.name
}

// TODO: Rewrite doc comment.
func (ts *Server) ServeLocal(cli *client.Client, opts ...Options) error {
	err := ts.init(opts...)
	if err != nil {
		return err
	}

	err = ts.ln.Init(ts)
	if err != nil {
		return err
	}

	host := cli.Config().Host
	port := uint16(cli.Config().Port)

	_, err = ts.ServeAddr(host, port, opts...)
	if err != nil {
		return err
	}

	// Attempt to connect with the user configuration.
	// Return if we are done, since we
	err = cli.Connect(client.WithLocalDialer())
	if err != nil {
		return err
	}

	return nil
}

// TODO: Rewrite doc comment.
// ServeAddr sets and start a gRPC teamserver listener (on MutualTLS) with registered
// teamserver services onto it.
// Starting listeners from application code (not from teamserver' commands) should most
// of the time be done with this function, as it will return you the gRPC server to which
// you can attach any application-specific APIs.
func (ts *Server) ServeAddr(host string, port uint16, opts ...Options) (net.Listener, error) {
	err := ts.init(opts...)
	if err != nil {
		return nil, err
	}

	listener, err := ts.ln.Listen(fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return listener, err
	}

	serv, err := ts.ln.Serve(listener)
	if err != nil {
		return listener, err
	}

	for _, hook := range ts.opts.hooks {
		if err := hook(serv); err != nil {
			return listener, err
		}
	}

	return listener, nil
}

// ServeDaemon is a blocking call which starts the server as daemon process, using
// either the provided host:port arguments, or the ones found in the teamserver config.
// It also accepts a function that will be called just after starting the server, so
// that users can still register their per-application services before actually blocking.
func (ts *Server) ServeDaemon(host string, port uint16, opts ...Options) error {
	log := ts.NamedLogger("daemon", "main")

	// cli args take president over config
	if host == blankHost {
		host = ts.opts.config.DaemonMode.Host
		log.Infof("No host specified, using config file default: %s", host)
	}
	if port == blankPort {
		port = uint16(ts.opts.config.DaemonMode.Port)
		log.Infof("No port specified, using config file default: %d", port)
	}

	log.Infof("Starting %s teamserver daemon on %s:%d ...", ts.Name(), host, port)
	ln, err := ts.ServeAddr(host, port, opts...)
	if err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Now that the main teamserver listener is started,
	// we can start all our persistent teamserver listeners.
	// That way, if any of them collides with our current bind,
	// we just serve it for him
	hostPort := regexp.MustCompile(fmt.Sprintf("%s:%d", host, port))

	err = ts.startPersistentListeners()
	if err != nil && hostPort.MatchString(err.Error()) {
		log.Warnf("Error starting persistent listeners: %s", err)
	}

	// TODO: Close server ? When ?
	done := make(chan bool)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	go func() {
		<-signals
		println("HERE")
		log.Infof("Received SIGTERM, exiting ...")
		ln.Close()
		done <- true
	}()
	<-done

	return nil
}

// Close gracefully stops all components of the server,
// letting pending connections to it to finish first.
// func (ts *Server) Close() {
// 	defer ts.log.Writer().Close()
// 	// defer ts.audit.Writer().Close()
// }

// NamedLogger returns a new logging "thread" which should grossly
// indicate the package/general domain, and a more precise flow/stream.
func (ts *Server) NamedLogger(pkg, stream string) *logrus.Entry {
	return ts.log.WithFields(logrus.Fields{
		log.PackageFieldKey: pkg,
		"stream":            stream,
	})
}

// init a function that must not be ran when the teamserver
// is instantiated, but when it starts serving its users.
// This starts database connections, certificates setup, applies last-minute options, etc.
func (ts *Server) init(opts ...Options) error {
	var err error

	ts.initOnce.Do(func() {
		// Database configuration.
		ts.opts.dbConfig = ts.getDatabaseConfig()

		// Last time for setting options.
		ts.apply(opts...)

		// Load any relevant server configuration: on disk,
		// contained in options, or the default one.
		ts.opts.config = ts.GetConfig()

		// Database
		if ts.opts.db == nil {
			dbLogger := ts.NamedLogger("database", "database")
			ts.db, err = db.NewClient(ts.opts.dbConfig, dbLogger)
			if err != nil {
				return
			}
		}

		// Certificate infrastructure
		certsLog := ts.NamedLogger("certs", "certificates")
		ts.certs = certs.NewManager(ts.db.Session(&gorm.Session{}), certsLog, ts.AppDir())
	})

	return err
}

// func (ts *Server[_]) newServer() *Server {
// 	serv := &Server{
// 		name:       ts.name,
// 		rootDirEnv: ts.rootDirEnv,
// 		log:        ts.log,
// 		// audit:                   ts.audit,
// 		opts:                    ts.opts,
// 		config:                  ts.config,
// 		certs:                   ts.certs,
// 		userTokens:              ts.userTokens,
// 		init:                    &sync.Once{},
// 		UnimplementedTeamServer: &proto.UnimplementedTeamServer{},
// 	}
//
// 	// One session per listener should be enough for now.
// 	serv.db = ts.db.Session(&gorm.Session{
// 		FullSaveAssociations: true,
// 	})
//
// 	return serv
// }
