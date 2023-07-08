package server

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/reeflective/team/internal/systemd"
	"github.com/reeflective/team/server"
	"github.com/spf13/cobra"
)

func daemoncmd(serv *server.Server) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		lhost, err := cmd.Flags().GetString("host")
		if err != nil {
			return fmt.Errorf("Failed to parse --host flag %s\n", err)
		}
		lport, err := cmd.Flags().GetUint16("port")
		if err != nil {
			return fmt.Errorf("Failed to parse --port flag %s\n", lport, err)
		}

		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic:\n%s", debug.Stack())
				fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
				os.Exit(99)
			}
		}()

		// Blocking call, your program will only exit/resume on Ctrl-C/SIGTERM
		return serv.ServeDaemon(lhost, lport)
	}
}

func startListenerCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		lhost, _ := cmd.Flags().GetString("host")
		lport, _ := cmd.Flags().GetUint16("port")
		persistent, _ := cmd.Flags().GetBool("persistent")

		_, err := serv.ServeAddr(lhost, lport)
		if err == nil {
			fmt.Printf(info+"Teamserver listener started on %s:%d\n", lhost, lport)
			if persistent {
				serv.AddListener(lhost, lport)
			}
		} else {
			fmt.Printf(warn+"Failed to start job %v\n", err)
		}
	}
}

func systemdConfigCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
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
			fmt.Printf(warn+"Failed to find teamserver daemon command in tree: %s", err)
		}

		config.Args = append(callerArgs(cmd.Parent()), daemonCmd.Name())
		if len(config.Args) > 0 && binPath != "" {
			config.Args[0] = binPath
		}

		systemdConfig := systemd.NewFrom(serv.Name(), config)
		fmt.Printf(systemdConfig)
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
