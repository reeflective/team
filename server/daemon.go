package server

import (
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/reeflective/team/internal/log"
)

var (
	blankHost = "-"
	blankPort = uint16(0)
)

// ServeDaemon is a blocking call which starts the server as daemon process, using
// either the provided host:port arguments, or the ones found in the teamserver config.
// It also accepts a function that will be called just after starting the server, so
// that users can still register their per-application services before actually blocking.
func (s *Server) ServeDaemon(host string, port uint16, postStart ...func(s *Server)) error {
	daemonLog := log.NamedLogger(s.log, "daemon", "main")

	// TODO: Use the logger stdout instead of printf ?

	// cli args take president over config
	if host == blankHost {
		daemonLog.Info("No cli lhost, using config file or default value")
		host = s.config.DaemonMode.Host
	}
	if port == blankPort {
		daemonLog.Info("No cli lport, using config file or default value")
		port = uint16(s.config.DaemonMode.Port)
	}

	daemonLog.Infof("Starting %s teamserver daemon %s:%d ...", s.Name(), host, port)
	_, ln, err := s.ServeAddr(host, port)
	if err != nil {
		return fmt.Errorf("failed to start daemon %w", err)
	}

	for _, startFunc := range postStart {
		startFunc(s)
	}

	// Now that the main teamserver listener is started,
	// we can start all our persistent teamserver listeners.
	// That way, if any of them collides with our current bind,
	// we just serve it for him
	hostPort := regexp.MustCompile(fmt.Sprintf("%s:%d", host, port))

	err = s.startPersistentJobs()
	if err != nil && hostPort.MatchString(err.Error()) {
		daemonLog.Infof("Error starting persistent listeners: %s", err)
	}

	done := make(chan bool)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	go func() {
		<-signals
		daemonLog.Infof("Received SIGTERM, exiting ...")
		ln.Close()
		done <- true
	}()
	<-done

	return nil
}
