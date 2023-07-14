package server

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
	"runtime/debug"
	"syscall"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/certs"
)

// Handler represents a teamserver listener stack.
// It must be satisfied by any type aiming to be used
// as a transport and RPC backend of the teamserver.
//
// The server type parameter represents the server-side of a connection
// between a teamclient and the teamserver. It may or may not offer the
// RPC services to the client yet. Implementations are free to decide.
// TODO: Write about the Options(hooks) to use with this generic type.
type Handler[server any] interface {
	Name() string
	Init(s *Server) error
	Listen(addr string) (ln net.Listener, err error)
	Serve(net.Listener) (serv server, err error)
	Close() error
}

func (ts *Server) ServeHandler(handler Handler[any], lnID, host string, port uint16, options ...Options) error {
	log := ts.NamedLogger("teamserver", "handler")

	// If server was not initialized yet, do it.
	err := ts.init(options...)
	if err != nil {
		return ts.errorf("%w: %w", ErrTeamServer, err)
	}

	// Let the handler initialize itself: load everything it needs from
	// the server, configuration, fetch certificates, log stuff, etc.
	err = handler.Init(ts)
	if err != nil {
		return ts.errorWith(log, "%w: %w", ErrListener, err)
	}

	// Now let the handler start listening on somewhere.
	laddr := fmt.Sprintf("%s:%d", host, port)

	listener, err := handler.Listen(laddr)
	if err != nil {
		return ts.errorWith(log, "%s: %w", ErrListener, err)
	}

	// The previous is not blocking, serve the listener immediately.
	serverConn, err := handler.Serve(listener)
	if err != nil {
		return ts.errorWith(log, "%w: %w", ErrListener, err)
	}

	// The server is running, so add a job anyway.
	ts.addListenerJob(lnID, host, int(port), handler)

	// Run provided server hooks on the server interface.
	// Any error arising from this is returned as is, for
	// users can directly compare it with their own errors.
	for _, hook := range ts.opts.hooks[handler.Name()] {
		if err := hook(serverConn); err != nil {
			return ts.errorWith(log, "%w: %w", ErrTeamServer, err)
		}
	}

	return nil
}

func (ts *Server) ServeAddr(name, host string, port uint16, opts ...Options) (jobID string, err error) {
	handler := ts.handlers[name]

	// The default handler can never be nil, as even the
	// default one is a pure fake in-memory teamclient.
	if handler == nil {
		handler = ts.self
	}

	// Generate the listener ID now so we can return it.
	listenerID := getRandomID()

	err = ts.ServeHandler(handler, listenerID, host, port, opts...)

	return listenerID, err
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
		log.Debugf("No host specified, using config file default: %s", host)
	}

	if port == blankPort {
		port = uint16(ts.opts.config.DaemonMode.Port)
		log.Debugf("No port specified, using config file default: %d", port)
	}

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("panic:\n%s", debug.Stack())
		}
	}()

	// Start the listener.
	log.Infof("Starting %s teamserver daemon on %s:%d ...", ts.Name(), host, port)

	listenerID, err := ts.ServeAddr(ts.self.Name(), host, port, opts...)
	if err != nil {
		return err
	}

	// Now that the main teamserver listener is started,
	// we can start all our persistent teamserver listeners.
	// That way, if any of them collides with our current bind,
	// we just serve it for him
	hostPort := regexp.MustCompile(fmt.Sprintf("%s:%d", host, port))

	err = ts.StartPersistentListeners(ts.opts.continueOnError)
	if err != nil && hostPort.MatchString(err.Error()) {
		log.Errorf("Error starting persistent listeners: %s", err)
	}

	done := make(chan bool)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)

	go func() {
		<-signals
		log.Infof("Received SIGTERM, exiting ...")
		ts.CloseListener(listenerID)
		done <- true
	}()
	<-done

	return nil
}

func (ts *Server) Serve(cli *client.Client, opts ...Options) error {
	host := cli.Config().Host
	port := uint16(cli.Config().Port)

	// Now let the handler start listening on somewhere.
	laddr := host
	if port != 0 {
		laddr = fmt.Sprintf("%s:%d", laddr, port)
	}

	if laddr == "" {
		laddr = "runtime"
	}

	if ts.self != nil {
		// Some errors might come from user-provided hooks,
		// so we don't wrap errors again, our own errors
		// have been prepared accordingly in this call.
		err := ts.ServeHandler(ts.self, "", host, port, opts...)
		if err != nil {
			return err
		}
	}

	// Attempt to connect with the user configuration.
	// Log the error by default, the client might not.
	err := cli.Connect(client.WithLocalDialer())
	if err != nil {
		return ts.errorf(err.Error())
	}

	return nil
}

// Handlers returns a copy of its teamserver handlers map.
func (ts *Server) Handlers() map[string]Handler[any] {
	handlers := make(map[string]Handler[any], len(ts.handlers))

	for name, handler := range ts.handlers {
		handlers[name] = handler
	}

	return handlers
}

// Close gracefully stops all components of the server,
// letting pending connections to it to finish first.
// func (ts *Server) Close() {
// 	defer ts.log().Writer().Close()
// 	// defer ts.audit.Writer().Close()
// }

func (ts *Server) init(opts ...Options) error {
	var err error

	// Always reaply options, since it could be used by different listeners.
	ts.apply(opts...)

	ts.initOnce.Do(func() {
		if err = ts.initDatabase(); err != nil {
			return
		}

		// Database configuration.
		// At creation time, we ensured that server had
		// a valid database configuration, but we might
		// // have been modified with options to Serve().
		// ts.opts.dbConfig, err = ts.getDatabaseConfig()
		// if err != nil {
		// 	err = ts.errorf("%w: %w", ErrDatabase, err)
		// 	return
		// }
		//
		// // Connect to database if not connected already.
		// if ts.db == nil {
		// 	dbLogger := ts.NamedLogger("database", "database")
		// 	ts.db, err = db.NewClient(ts.opts.dbConfig, dbLogger)
		// 	if err != nil {
		// 		err = ts.errorf("%w: %w", ErrDatabase, err)
		// 		return
		// 	}
		// }

		// Load any relevant server configuration: on disk,
		// contained in options, or the default one.
		ts.opts.config = ts.GetConfig()

		// Certificate infrastructure, will make the code panic if unable to work properly.
		certsLog := ts.NamedLogger("certs", "certificates")
		ts.certs = certs.NewManager(ts.fs, ts.dbSession(), certsLog, ts.Name(), ts.TeamDir())
	})

	return err
}