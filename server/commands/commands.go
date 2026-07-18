package commands

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"fmt"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/reeflective/team/client"
	cli "github.com/reeflective/team/client/commands"
	"github.com/reeflective/team/internal/command"
	"github.com/reeflective/team/log"
	"github.com/reeflective/team/server"
)

// Generate returns a "teamserver" command root and its tree for teamserver (server-side) management.
// It requires a teamclient so as to bind its "teamclient" tree as a subcommand of the server root.
// This is so that all CLI applications which can be a teamserver can also be a client of their own.
//
// ** Commands do:
//   - Work even if the teamserver/client returns errors: those are returned &| printed &| logged.
//   - Use the cobra utilities OutOrStdout(), ErrOrStdErr(), ... for all and every command output.
//   - Have attached completions for users/listeners/config files of all sorts, and other things.
//   - Have the ability to be ran in closed-loop console applications ("single runtime shell").
//
// ** Commands do NOT:
//   - Ensure they are connected to a server instance before running (in memory).
//   - Call os.Exit() anywhere, thus will not exit the program embedding them.
//   - Ignite/start the teamserver core/filesystem/backends before they absolutely need to.
//     Consequently, do not touch the filesystem until they absolutely need to.
//   - Connect the client more than once to the teamserver.
//   - Start persistent listeners, excluding the daemon command.
func Generate(teamserver *server.Server, teamclient *client.Client) *cobra.Command {
	// Server-only commands always need to have open log
	// files, most of the time access to the database, etc.
	// On top, they need a listener in memory.
	servCmds := serverCommands(teamserver, teamclient)

	// We bind the same runners to the client-side commands.
	cliCmds := cli.Generate(teamclient)
	cliCmds.Use = "client"
	cliCmds.GroupID = command.TeamServerGroup

	servCmds.AddCommand(cliCmds)

	return servCmds
}

func serverCommands(server *server.Server, client *client.Client) *cobra.Command {
	name := server.Name()
	teamCmd := &cobra.Command{
		Use:   "teamserver",
		Short: fmt.Sprintf("Manage the %s teamserver and users", name),
		Long: fmt.Sprintf(`Manage the %[1]s teamserver: users, listeners, and the connection configs
operators use to reach it.

The teamserver is embedded in %[1]s; these commands administer it. A typical
bring-up is:

  1. Create users     teamserver user --name alice --host <bind>
  2. Add listeners    teamserver listen --host <bind> --persistent
  3. Run the server   teamserver daemon --host <bind> --port <port>

Each 'user' command writes a *.teamclient.cfg file to hand to that operator; they
import it with 'teamserver client import' (or a client-only binary's 'import'). The
teamserver only AUTHENTICATES users (proves identity); what a user is allowed to do
is decided by %[1]s itself.

Client-side commands (import a config, list users, show version) live under the
'client' subcommand. Run 'teamserver guide' for a fuller walkthrough.`, name),
		SilenceUsage: true,
	}

	// Groups
	teamCmd.AddGroup(
		&cobra.Group{ID: command.TeamServerGroup, Title: command.TeamServerGroup},
		&cobra.Group{ID: command.UserManagementGroup, Title: command.UserManagementGroup},
	)

	teamFlags := pflag.NewFlagSet("teamserver", pflag.ContinueOnError)
	teamFlags.CountP("verbosity", "v", "Increase stdout log verbosity; repeat to go louder (-v, -vv, -vvv)")
	teamFlags.String("log-format", "", "console log format (console, text, json)")
	teamCmd.PersistentFlags().AddFlagSet(teamFlags)

	// Apply the chosen console log format (console/text/json) before running.
	teamCmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		if format, _ := cmd.Flags().GetString("log-format"); format != "" {
			server.SetLogFormat(log.Format(format))
		}

		return nil
	}

	carapace.Gen(teamCmd).FlagCompletion(carapace.ActionMap{
		"log-format": command.LogFormatCompleter(),
	})

	// [ Listeners and servers control commands ] ------------------------------------------

	// Start a listener
	listenCmd := &cobra.Command{
		Use:   "listen",
		Short: "Start a teamserver listener (non-blocking)",
		Long: `Start a listener (a bind job) for a registered transport stack, without blocking.
Use --persistent to save it so 'daemon' restarts it automatically. Pick a non-default
transport with --listener (completed by stack name).`,
		Example: `  # Default stack on localhost, remembered across restarts
  teamserver listen --host localhost --persistent

  # A specific stack on another interface/port
  teamserver listen --host 10.0.0.5 --port 32333 --listener gRPC --persistent`,
		GroupID: command.TeamServerGroup,
		RunE:    startListenerCmd(server),
	}

	lnFlags := pflag.NewFlagSet("listener", pflag.ContinueOnError)
	lnFlags.StringP("host", "H", "", "interface to bind server to")
	lnFlags.StringP("listener", "l", "", "listener stack to use instead of default (completed)")
	lnFlags.Uint16P("port", "P", 31337, "tcp listen port")
	lnFlags.BoolP("persistent", "p", false, "make listener persistent across restarts")
	listenCmd.Flags().AddFlagSet(lnFlags)

	listenComps := make(carapace.ActionMap)
	listenComps["host"] = interfacesCompleter()
	listenComps["listener"] = carapace.ActionCallback(listenerTypeCompleter(client, server))
	carapace.Gen(listenCmd).FlagCompletion(listenComps)

	teamCmd.AddCommand(listenCmd)

	// Close a listener
	closeCmd := &cobra.Command{
		Use:   "close",
		Short: "Close a listener and remove it from persistent ones if it's one",
		Long: `Close one or more running listeners by ID (a unique prefix is enough) and remove
them from the saved/persistent set. IDs are shown by 'status' and are completed.`,
		Example: `  teamserver close 3f9ab21c
  teamserver close 3f9ab21c 8c1de490`,
		Args:    cobra.MinimumNArgs(1),
		GroupID: command.TeamServerGroup,
		Run:     closeCmd(server),
	}

	closeComps := carapace.Gen(closeCmd)
	closeComps.PositionalAnyCompletion(carapace.ActionCallback(listenerIDCompleter(client, server)))

	closeComps.PreRun(func(cmd *cobra.Command, args []string) {
		if cmd.PersistentPreRunE != nil {
			cmd.PersistentPreRunE(cmd, args)
		}

		if cmd.PreRunE != nil {
			cmd.PreRunE(cmd, args)
		}
	})

	teamCmd.AddCommand(closeCmd)

	// Daemon (blocking listener and persistent jobs)
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Start the teamserver in daemon mode (blocking)",
		Long: `Run the teamserver in the foreground (blocking) until SIGTERM / Ctrl-C. Starts the
main listener plus every persistent listener. With no --host/--port, the values from
the teamserver config are used.`,
		Example: `  teamserver daemon --host 0.0.0.0 --port 31337
  teamserver daemon                         # use config defaults`,
		GroupID: command.TeamServerGroup,
		RunE:    daemoncmd(server),
	}
	daemonCmd.Flags().StringP("host", "l", "-", "multiplayer listener host")
	daemonCmd.Flags().Uint16P("port", "p", uint16(0), "multiplayer listener port")

	daemonComps := make(carapace.ActionMap)
	daemonComps["host"] = interfacesCompleter()
	carapace.Gen(daemonCmd).FlagCompletion(daemonComps)

	teamCmd.AddCommand(daemonCmd)

	// Systemd configuration output
	systemdCmd := &cobra.Command{
		Use:   "systemd",
		Short: "Print a systemd unit file for the application teamserver, with options",
		Long: `Render a systemd unit that runs 'teamserver daemon'. Prints to stdout unless --save
is given. --user sets the OS user the service runs as (a value, e.g. --user myapp),
--binpath the executable path baked into the unit.`,
		Example: `  teamserver systemd --binpath /usr/local/bin/myapp --host 0.0.0.0 --port 31337
  teamserver systemd --user myapp --save /etc/systemd/system/myapp.service`,
		GroupID: command.TeamServerGroup,
		RunE:    systemdConfigCmd(server),
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
	sComps["host"] = interfacesCompleter()
	carapace.Gen(systemdCmd).FlagCompletion(sComps)

	teamCmd.AddCommand(systemdCmd)

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show the status of the teamserver (listeners, configurations, health...)",
		Long: `Show the teamserver's home directory, database, config path, log files and levels,
certificate files, and the state of all listeners (running and saved/persistent).`,
		GroupID: command.TeamServerGroup,
		Run:     statusCmd(server),
	}

	teamCmd.AddCommand(statusCmd)

	// [ Users and data control commands ] -------------------------------------------------

	// Add user
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "Create a user for this teamserver and generate its client configuration file",
		Long: `Create a user and generate its connection config (*.teamclient.cfg): a client
certificate and API token the operator uses to authenticate. The file is written to
the current directory unless --save <dir> is given, or --system (use the current OS
user and save into this app's client configs directory).`,
		Example: `  teamserver user --name alice --host teamserver.example.com
  teamserver user --name bob   --host 10.0.0.5 --port 32333 --save ~/handout/
  teamserver user --system`,
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
	userComps["host"] = interfacesCompleter()
	carapace.Gen(userCmd).FlagCompletion(userComps)

	// Delete and kick user
	rmUserCmd := &cobra.Command{
		Use:   "delete",
		Short: "Remove a user from the teamserver, and revoke all its current tokens",
		Long: `Delete a user and its cryptographic material. This takes effect immediately: the
user's live sessions are refused on their next request and its TLS credentials stop
working.`,
		Example: `  teamserver delete alice`,
		GroupID: command.UserManagementGroup,
		Args:    cobra.ExactArgs(1),
		Run:     rmUserCmd(server),
	}

	teamCmd.AddCommand(rmUserCmd)

	rmUserComps := carapace.Gen(rmUserCmd)

	rmUserComps.PositionalCompletion(carapace.ActionCallback(userCompleter(server)))

	rmUserComps.PreRun(func(cmd *cobra.Command, args []string) {
		if cmd.PersistentPreRunE != nil {
			cmd.PersistentPreRunE(cmd, args)
		}

		if cmd.PreRunE != nil {
			cmd.PreRunE(cmd, args)
		}
	})

	// Import a list of users and their credentials.
	cmdImportCA := &cobra.Command{
		Use:   "import",
		Short: "Import a certificate Authority file containing teamserver users",
		Long: `Import a users Certificate Authority exported by another teamserver, adding its
users to this one. The file is JSON of the form {"certificate":"...","private_key":"..."}.`,
		Example: `  teamserver import ~/.other_app/teamserver/certs/other_app_user-ca-cert.teamserver.pem`,
		GroupID: command.UserManagementGroup,
		Args:    cobra.ExactArgs(1),
		Run:     importCACmd(server),
	}

	iComps := carapace.Gen(cmdImportCA)
	iComps.PositionalCompletion(
		carapace.Batch(
			carapace.ActionCallback(cli.ConfigsCompleter(client, "teamserver/certs", ".teamserver.pem", "other teamservers user CAs", true)),
			carapace.ActionFiles().Tag("teamserver user CAs"),
		).ToA(),
	)

	teamCmd.AddCommand(cmdImportCA)

	// Export the list of users and their credentials.
	cmdExportCA := &cobra.Command{
		Use:   "export",
		Short: "Export a Certificate Authority file containing the teamserver users",
		Long: `Export this teamserver's users CA (all users) to a file, so another teamserver can
import and trust the same operators. Writes to the current directory when no path is
given.`,
		Example: `  teamserver export ~/myapp-users.teamserver.ca`,
		GroupID: command.UserManagementGroup,
		Args:    cobra.RangeArgs(0, 1),
		Run:     exportCACmd(server),
	}

	carapace.Gen(cmdExportCA).PositionalCompletion(carapace.ActionFiles())
	teamCmd.AddCommand(cmdExportCA)

	// [ Holistic help ] -------------------------------------------------------------------

	// A cobra "additional help topic" (no Run): a single walkthrough that stays out of
	// the command list but is discoverable via '<app> teamserver guide'.
	guideCmd := &cobra.Command{
		Use:   "guide",
		Short: "In-depth guide: user lifecycle, listeners, daemon/systemd, revocation",
		Long: fmt.Sprintf(`%[1]s teamserver — operator guide

The teamserver is embedded directly in %[1]s. There is no separate server to install:
the same binary authenticates operators, serves listeners, and (as a client) connects
to them.

1. Users and configs
   Create one config per operator:
       teamserver user --name alice --host <bind-address>
   This writes alice's *.teamclient.cfg (a client certificate + API token) to the
   current directory (or --save <dir>, or --system for the current OS user). Hand that
   file to the operator; they import it with:
       teamserver client import alice.teamclient.cfg
   The teamserver only proves WHO an operator is. Authorization — what each operator
   may do — is entirely up to %[1]s.

2. Listeners
   A listener is a bind job for a transport stack. Start one without blocking:
       teamserver listen --host <bind> --port <port> --persistent
   --persistent saves it so the daemon restarts it automatically. Inspect and close
   listeners with:
       teamserver status
       teamserver close <id-prefix>

3. Running as a service
   Run in the foreground (starts the main + all persistent listeners):
       teamserver daemon --host <bind> --port <port>
   Generate a systemd unit for it:
       teamserver systemd --binpath $(which %[1]s) --save unit.service

4. Sharing users between servers
   Export the users CA and import it on another teamserver so both trust the same
   operators:
       teamserver export users.ca
       teamserver import users.ca

5. Revoking access
   Delete a user; its live sessions are refused on the next request and its TLS
   credentials stop working immediately:
       teamserver delete alice

Shell completion:
    source <(teamserver _carapace <shell>)`, name),
	}

	teamCmd.AddCommand(guideCmd)

	return teamCmd
}
