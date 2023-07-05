package server

import (
	"context"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/reeflective/team/client"
	cli "github.com/reeflective/team/command/client"
	"github.com/reeflective/team/internal/command"
	"github.com/reeflective/team/internal/proto"
	"github.com/reeflective/team/server"
)

const (
	// ANSI Colors
	normal    = "\033[0m"
	black     = "\033[30m"
	red       = "\033[31m"
	green     = "\033[32m"
	orange    = "\033[33m"
	blue      = "\033[34m"
	purple    = "\033[35m"
	cyan      = "\033[36m"
	gray      = "\033[37m"
	bold      = "\033[1m"
	clearln   = "\r\x1b[2K"
	upN       = "\033[%dA"
	downN     = "\033[%dB"
	underline = "\033[4m"

	// info - Display colorful information
	info = bold + cyan + "[*] " + normal
	// warn - warn a user
	warn = bold + red + "[!] " + normal
	// debugl - Display debugl information
	debugl = bold + purple + "[-] " + normal
)

// Commands returns a "teamserver" command root and its tree for teamserver (server-side) management.
// It requires a teamclient so as to bind its "teamclient" tree as a subcommand of the server root.
// This is so that all CLI applications which can be a teamserver can also be a client of their own.
func Commands(teamserver *server.Server, teamclient *client.Client) *cobra.Command {
	serveAndConnect := func(cmd *cobra.Command, args []string) error {
		// If the server is already serving us with an in-memory con, return.
		// Also, the daemon command does not need a teamclient connection.
		if teamclient.IsConnected() {
			return nil
		}

		// And connect the client locally, only needed.
		_, err := teamserver.Serve(teamclient)

		// And connect the client locally, only needed.
		err = teamclient.Connect()
		if err != nil {
			return err
		}

		return nil
	}

	// Server-only commands always need to have open log
	// files, most of the time access to the database, etc.
	// On top, they need a listener in memory.
	servCmds := serverCommands(teamserver, teamclient)

	for _, cmd := range servCmds.Commands() {
		cmd.PersistentPreRunE = serveAndConnect
	}

	// We bind the same runners to the client-side commands.
	cliCmds := cli.Commands(teamclient)
	cliCmds.Use = "client"

	for _, cmd := range cliCmds.Commands() {
		cmd.PersistentPreRunE = serveAndConnect
	}

	servCmds.AddCommand(cliCmds)

	return servCmds
}

// PreRun returns a cobra command runner which connects the local teamclient to itself.
// If the client is connected, nothing happens and its current connection reused, which
// makes this runner able to be ran in closed-loop consoles.
func PreRun(teamserver *server.Server, teamclient *client.Client) command.CobraRunnerE {
	return func(cmd *cobra.Command, args []string) error {
		// If the server is already serving us with an in-memory con, return.
		// Also, the daemon command does not need a teamclient connection.
		if teamclient.IsConnected() {
			return nil
		}

		// And connect the client locally, only needed.
		_, err := teamserver.Serve(teamclient)

		// And connect the client locally, only needed.
		err = teamclient.Connect()
		if err != nil {
			return err
		}

		return nil
	}
}

// PostRun returns a cobra command runner which currently does nothing.
func PostRun(server *server.Server, client *client.Client) command.CobraRunnerE {
	return nil
}

func serverCommands(server *server.Server, client *client.Client) *cobra.Command {
	teamCmd := &cobra.Command{
		Use:   "teamserver",
		Short: "Manage the application server-side teamserver and users",
	}

	// Groups
	teamCmd.AddGroup(
		&cobra.Group{ID: command.TeamServerGroup, Title: command.TeamServerGroup},
		&cobra.Group{ID: command.UserManagementGroup, Title: command.UserManagementGroup},
	)

	// [ Listeners and servers control commands ] ------------------------------------------

	listenCmd := &cobra.Command{
		Use:     "listen",
		Short:   "Start a teamserver gRPC listener job (non-blocking)",
		GroupID: command.TeamServerGroup,
		Run:     startListenerCmd(server),
	}

	lnFlags := pflag.NewFlagSet("listener", pflag.ContinueOnError)
	lnFlags.StringP("host", "L", "", "interface to bind server to")
	lnFlags.Uint16P("port", "l", 31337, "tcp listen port")
	lnFlags.BoolP("persistent", "p", false, "make listener persistent across restarts")
	listenCmd.Flags().AddFlagSet(lnFlags)

	teamCmd.AddCommand(listenCmd)

	// systemd
	daemonCmd := &cobra.Command{
		Use:     "daemon",
		Short:   "Start the teamserver in daemon mode (blocking)",
		GroupID: command.TeamServerGroup,
		Run:     daemoncmd(server),
	}
	daemonCmd.Flags().StringP("host", "l", "-", "multiplayer listener host")
	daemonCmd.Flags().Uint16P("port", "p", uint16(0), "multiplayer listener port")

	teamCmd.AddCommand(daemonCmd)

	systemdCmd := &cobra.Command{
		Use:     "systemd",
		Short:   "Print a systemd unit file for the application teamserver, with options",
		GroupID: command.TeamServerGroup,
		Run:     systemdConfigCmd(server),
	}

	sFlags := pflag.NewFlagSet("systemd", pflag.ContinueOnError)
	sFlags.StringP("binpath", "b", "", "Specify the path of the teamserver application binary")
	sFlags.StringP("user", "u", "", "Specify the user for the systemd file to run with")
	sFlags.StringP("save", "s", "", "Directory/file in which to save config, instead of stdout")
	sFlags.StringP("host", "l", "", "Listen host to use in the systemd command line")
	sFlags.Uint16P("port", "p", 0, "Listen port in the systemd command line")
	systemdCmd.Flags().AddFlagSet(sFlags)

	sComps := make(carapace.ActionMap)
	sComps["save"] = carapace.ActionFiles()
	sComps["binpath"] = carapace.ActionFiles()
	carapace.Gen(systemdCmd).FlagCompletion(sComps)

	teamCmd.AddCommand(systemdCmd)

	// [ Users and data control commands ] -------------------------------------------------

	// Add user
	userCmd := &cobra.Command{
		Use:     "user",
		Short:   "Create a user for this teamserver and generate its client configuration file",
		GroupID: command.UserManagementGroup,
		Run:     createUserCmd(server, client),
	}

	teamCmd.AddCommand(userCmd)

	userFlags := pflag.NewFlagSet("user", pflag.ContinueOnError)
	userFlags.StringP("host", "l", "", "listen host")
	userFlags.Uint16P("port", "p", 0, "listen port")
	userFlags.StringP("save", "s", "", "directory/file in which to save config")
	userFlags.StringP("name", "n", "", "user name")
	userFlags.BoolP("system", "U", false, "Use the current OS user, and save its configuration directly in client dir")
	userCmd.Flags().AddFlagSet(userFlags)

	userComps := make(carapace.ActionMap)
	userComps["save"] = carapace.ActionDirectories()
	carapace.Gen(userCmd).FlagCompletion(userComps)

	// Delete and kick user
	rmUserCmd := &cobra.Command{
		Use:     "delete",
		Short:   "Remove a user from the teamserver, and revoke all its current tokens",
		GroupID: command.UserManagementGroup,
		Args:    cobra.ExactArgs(1),
		Run:     rmUserCmd(server),
	}

	teamCmd.AddCommand(rmUserCmd)

	carapace.Gen(rmUserCmd).PositionalCompletion(
		carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			req, err := server.GetUsers(context.Background(), &proto.Empty{})
			if err != nil {
				return carapace.ActionMessage("failed to get teamserver users: %s", err)
			}

			users := req.GetUsers()

			results := make([]string, len(users))
			for _, user := range users {
				results = append(results, strings.TrimSpace(user.Name))
			}

			return carapace.ActionValues(results...).Tag("teamserver users")
		}))

	// Import a list of users and their credentials.
	cmdImportCA := &cobra.Command{
		Use:     "import",
		Short:   "Import a certificate Authority file containing teamserver users",
		GroupID: command.UserManagementGroup,
		Args:    cobra.ExactArgs(1),
		Run:     importCACmd(server),
	}

	carapace.Gen(cmdImportCA).PositionalCompletion(carapace.ActionFiles())
	teamCmd.AddCommand(cmdImportCA)

	// Export the list of users and their credentials.
	cmdExportCA := &cobra.Command{
		Use:     "export",
		Short:   "Export a Certificate Authority file containing the teamserver users",
		GroupID: command.UserManagementGroup,
		Args:    cobra.RangeArgs(0, 1),
		Run:     exportCACmd(server),
	}

	carapace.Gen(cmdExportCA).PositionalCompletion(carapace.ActionFiles())
	teamCmd.AddCommand(cmdExportCA)

	return teamCmd
}
