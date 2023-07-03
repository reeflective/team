package server

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var (
	blankHost = "-"
	blankPort = uint16(0)
)

// ServeDaemon is a blocking call which starts the server as daemon process, using
// either the provided host:port arguments, or the ones found in the teamserver config.
// It also accepts a function that will be called just after starting the server, so
// that users can still register their per-application services before actually blocking.
func (s *Server) ServeDaemon(host string, port uint16, postStart func(s *Server)) {
	daemonLog := s.NamedLogger("daemon", "main")

	// cli args take president over config
	if host == blankHost {
		daemonLog.Info("No cli lhost, using config file or default value")
		host = s.config.DaemonConfig.Host
	}
	if port == blankPort {
		daemonLog.Info("No cli lport, using config file or default value")
		port = uint16(s.config.DaemonConfig.Port)
	}

	daemonLog.Infof("Starting Sliver daemon %s:%d ...", host, port)
	_, ln, err := s.ServeAddr(host, port)
	if err != nil {
		fmt.Printf("[!] Failed to start daemon %s", err)
		daemonLog.Errorf("Error starting client listener %s", err)
		os.Exit(1)
	}

	if postStart != nil {
		postStart(s)
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
}
