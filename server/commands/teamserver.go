package commands

import (
	"fmt"
	"runtime/debug"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/reeflective/team/internal/command"
	"github.com/reeflective/team/internal/systemd"
	"github.com/reeflective/team/server"
)

func daemoncmd(serv *server.Server) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}

		lhost, err := cmd.Flags().GetString("host")
		if err != nil {
			return fmt.Errorf("Failed to get --host flag: %s", err)
		}
		lport, err := cmd.Flags().GetUint16("port")
		if err != nil {
			return fmt.Errorf("Failed to get --port (%d) flag: %s", lport, err)
		}

		// Also written to logs in the teamserver code.
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "stacktrace from panic: \n"+string(debug.Stack()))
			}
		}()

		// Blocking call, your program will only exit/resume on Ctrl-C/SIGTERM
		return serv.ServeDaemon(lhost, lport)
	}
}

func startListenerCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}

		lhost, _ := cmd.Flags().GetString("host")
		lport, _ := cmd.Flags().GetUint16("port")
		persistent, _ := cmd.Flags().GetBool("persistent")

		_, err := serv.ServeAddr(lhost, lport)
		if err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), command.Info+"Teamserver listener started on %s:%d\n", lhost, lport)
			if persistent {
				serv.AddListener(lhost, lport)
			}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), command.Warn+"Failed to start job %v\n", err)
		}
	}
}

func closeCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}
	}
}

func systemdConfigCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}

		config := systemd.NewDefaultConfig()

		userf, _ := cmd.Flags().GetString("user")
		if userf != "" {
			config.User = userf
		}

		binPath, _ := cmd.Flags().GetString("binpath")
		if binPath != "" {
			config.Binpath = binPath
		}

		// The last argument is the systemd command:
		// its parent is the teamserver one, to which
		// should be attached the daemon command.
		daemonCmd, _, err := cmd.Parent().Find([]string{"daemon"})
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), command.Warn+"Failed to find teamserver daemon command in tree: %s", err)
		}

		config.Args = append(callerArgs(cmd.Parent()), daemonCmd.Name())
		if len(config.Args) > 0 && binPath != "" {
			config.Args[0] = binPath
		}

		systemdConfig := systemd.NewFrom(serv.Name(), config)
		fmt.Fprintf(cmd.OutOrStdout(), systemdConfig)
	}
}

func statusCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}
	}
}

func callerArgs(cmd *cobra.Command) []string {
	var args []string

	if cmd.HasParent() {
		args = callerArgs(cmd.Parent())
	}

	args = append(args, cmd.Name())

	return args
}
