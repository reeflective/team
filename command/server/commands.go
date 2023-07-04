package server

import (
	"strings"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/reeflective/team/client"
	cli "github.com/reeflective/team/command/client"
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

// Commands initliazes and returns a command tree to embed in the teamserver binary.
// It requires the server itself to use its functions.
func Commands(server *server.Server) *cobra.Command {
	teamCmd := &cobra.Command{
		Use:   "teamserver",
		Short: "Manage the application server-side teamserver and users",
	}

	// Groups
	teamCmd.AddGroup(
		&cobra.Group{ID: cli.TeamServerGroup, Title: cli.TeamServerGroup},
		&cobra.Group{ID: cli.UserManagementGroup, Title: cli.UserManagementGroup},
	)

	// [ Listeners and servers control commands ] ------------------------------------------

	listenCmd := &cobra.Command{
		Use:     "listen",
		Short:   "Start a teamserver gRPC listener job (non-blocking)",
		GroupID: cli.TeamServerGroup,
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
		GroupID: cli.TeamServerGroup,
		Run:     daemoncmd(server),
	}
	daemonCmd.Flags().StringP("host", "l", "-", "multiplayer listener host")
	daemonCmd.Flags().Uint16P("port", "p", uint16(0), "multiplayer listener port")

	teamCmd.AddCommand(daemonCmd)

	systemdCmd := &cobra.Command{
		Use:     "systemd",
		Short:   "Print a systemd unit file for the application teamserver, with options",
		GroupID: cli.TeamServerGroup,
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
		GroupID: cli.UserManagementGroup,
		Run:     createUserCmd(server),
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
		GroupID: cli.UserManagementGroup,
		Args:    cobra.ExactArgs(1),
		Run:     rmUserCmd(server),
	}

	teamCmd.AddCommand(rmUserCmd)

	carapace.Gen(rmUserCmd).PositionalCompletion(
		carapace.ActionCallback(func(c carapace.Context) carapace.Action {
			users, err := server.UserAll()
			if err != nil {
				return carapace.ActionMessage("failed to get teamserver users: %s", err)
			}

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
		GroupID: cli.UserManagementGroup,
		Args:    cobra.ExactArgs(1),
		Run:     importCACmd(server),
	}

	carapace.Gen(cmdImportCA).PositionalCompletion(carapace.ActionFiles())
	teamCmd.AddCommand(cmdImportCA)

	// Export the list of users and their credentials.
	cmdExportCA := &cobra.Command{
		Use:     "export",
		Short:   "Export a Certificate Authority file containing the teamserver users",
		GroupID: cli.UserManagementGroup,
		Args:    cobra.RangeArgs(0, 1),
		Run:     exportCACmd(server),
	}

	carapace.Gen(cmdExportCA).PositionalCompletion(carapace.ActionFiles())
	teamCmd.AddCommand(cmdExportCA)

	return teamCmd
}

// ConnectLocal returns the teamserver-only and teamclient command trees, with pre-runners configured to self-serve/connect
// to the binary in memory.
func ConnectLocal(teamserver *server.Server, teamclient *client.Client) (servCmds, cliCmds *cobra.Command) {
	serveAndConnect := func(cmd *cobra.Command, args []string) error {
		// If the server is already serving us with an in-memory con, return.
		// Also, the daemon command does not need a teamclient connection.
		if teamclient.IsConnected() {
			return nil
		}

		// We generate a local client (and its command tree) from our
		// server, and bind these client commands to our server tree.
		conn, _, err := teamserver.ServeLocal()
		if err != nil {
			return err
		}

		// And connect the client locally, only needed.
		err = teamclient.Connect(client.WithConnection(conn))
		if err != nil {
			return err
		}

		return nil
	}

	// Server-only commands always need to have open log
	// files, most of the time access to the database, etc.
	// On top, they need a listener in memory.
	servCmds = Commands(teamserver)

	for _, cmd := range servCmds.Commands() {
		cmd.PersistentPreRunE = serveAndConnect
	}

	// We bind the same runners to the client-side commands.
	cliCmds = cli.Commands(teamclient)

	for _, cmd := range cliCmds.Commands() {
		cmd.PersistentPreRunE = serveAndConnect
	}

	return servCmds, cliCmds
}

// ConnectUser is similar to SelfConnect: it returns the teamserver and teamclient command trees, but this time
// configured to connect to a remote server instance if one for this application is detected, even if our binary
// is the server. This in effect tells the server: "we already have a server running, you're just a client here".
//
// Detailed workflow and steps:
// 1. The binary starts, the application looks for its default user config (~/.app/configs/app_user_default.cfg).
// 2 - If found, check if there is another application server running. If yes, connect to it remotely.
// 3 - If no user default config for this app is found, start locally, don't attempt to look at other configs.
func ConnectUser(teamserver *server.Server, teamclient *client.Client) (servCmds, cliCmds *cobra.Command) {
	serveAndConnect := func(cmd *cobra.Command, args []string) error {
		// If the server is already serving us, just return, because
		// we might be called repeatedly from a closed-loop console,
		// with any number of invocations in the same runtime.
		if teamclient.IsConnected() {
			return nil
		}

		// Run the server either as a remote client connected to either
		// the default running sister server, or against ourselves if needed.
		_, err := teamserver.Serve(teamclient, server.WithOSUserDefault())

		return err
	}

	// Server-only commands always need to have open log
	// files, most of the time access to the database, etc.
	// On top, they need a listener in memory.
	servCmds = Commands(teamserver)

	for _, cmd := range servCmds.Commands() {
		cmd.PersistentPreRunE = serveAndConnect
	}

	// We bind the same runners to the client-side commands.
	cliCmds = cli.Commands(teamclient)

	for _, cmd := range cliCmds.Commands() {
		cmd.PersistentPreRunE = serveAndConnect
	}

	return
}

// ServeRun returns a cobra command connecting the client to the teamserver.
// This should generally be used as one of (or part of another) command pre-runner.
func ServeRun(hooks ...func(serv *server.Server)) func(cmd *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		// if cli.IsConnected() {
		// 	return nil
		// }
		//
		// if err := cli.Connect(); err != nil {
		// 	fmt.Printf(warn+"Error connecting to teamserver: %s\n", err)
		// 	return err
		// }

		return nil
	}
}

// CloseRun returns a cobra command disconnecting the client from the teamserver.
// This should generally be used as one of (or part of another) command post-runner.
func CloseRun(hooks ...func(serv *server.Server)) func(cmd *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		// THis is safe, client ensures to close what can be.
		// cli.Disconnect()
		return nil
	}
}
