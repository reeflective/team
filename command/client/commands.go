package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/rsteube/carapace/pkg/style"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/internal/command"
	"github.com/reeflective/team/internal/version"
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

// Commands initliazes and returns a command tree to embed in client applications
// connecting to a teamserver. It requires only the client to use its functions.
func Commands(cli *client.Client) *cobra.Command {
	clientCmds := clientCommands(cli)

	for _, cmd := range clientCmds.Commands() {
		if isNoConnect(cmd) {
			continue
		}
		cmd.PersistentPreRunE = PreRun(cli)
	}

	return clientCmds
}

// PreRun returns a cobra command runner which connects the client to its teamserver.
// If the client is connected, nothing happens and its current connection reused, which
// makes this runner able to be ran in closed-loop consoles.
func PreRun(teamclient *client.Client) command.CobraRunnerE {
	return func(cmd *cobra.Command, args []string) error {
		// If the server is already serving us with an in-memory con, return.
		// Also, the daemon command does not need a teamclient connection.
		if teamclient.IsConnected() {
			return nil
		}

		// And connect the client locally, only needed.
		return teamclient.Connect()
	}
}

// PostRun is a cobra command runner that disconnects the client from its server.
// It does so unconditionally, so this is not suited for being included in consoles.
func PostRun(client *client.Client) command.CobraRunnerE {
	return func(cmd *cobra.Command, _ []string) error {
		client.Disconnect()
		return nil
	}
}

func clientCommands(cli *client.Client) *cobra.Command {
	teamCmd := &cobra.Command{
		Use:     "teamclient",
		Short:   "Client-only teamserver commands (import configs, show users, etc)",
		GroupID: command.TeamServerGroup,
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print teamserver client version",
		Run: func(cmd *cobra.Command, args []string) {
			// Server first
			serverVer, err := cli.ServerVersion()
			if err != nil {
				fmt.Printf(warn+"Server (error: %w)\r\n", err)
			}
			// if serverVer == nil {
			// fmt.Printf(warn+"No server version info)\r\n", err)
			// return
			// }

			dirty := ""
			if serverVer.Dirty {
				dirty = fmt.Sprintf(" - %sDirty%s", bold, normal)
			}
			serverSemVer := fmt.Sprintf("%d.%d.%d", serverVer.Major, serverVer.Minor, serverVer.Patch)
			fmt.Printf(info+"Server v%s - %s%s\n", serverSemVer, serverVer.Commit, dirty)

			// Client
			fmt.Printf(info+"Client %s\n", version.Full())
		},
	}

	teamCmd.AddCommand(versionCmd)

	importCmd := &cobra.Command{
		Use:   "import",
		Short: fmt.Sprintf("Import a teamserver client configuration file for %s", cli.Name()),
		Run: func(cmd *cobra.Command, args []string) {
			if 0 < len(args) {
				for _, arg := range args {
					conf, err := cli.ReadConfig(arg)
					if err != nil {
						fmt.Printf("[!] %s\n", err)
						os.Exit(3)
					}
					cli.SaveConfig(conf)
				}
			} else {
				fmt.Printf("Missing config file path, see --help")
			}
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{}, cobra.ShellCompDirectiveDefault
		},
	}

	iFlags := pflag.NewFlagSet("import", pflag.ContinueOnError)
	iFlags.BoolP("default", "d", false, "Set this config as the default one, if no default config exists already.")
	importCmd.Flags().AddFlagSet(iFlags)

	iComps := carapace.Gen(importCmd)
	iComps.PositionalCompletion(
		carapace.Batch(
			carapace.ActionCallback(teamserversCompleter(cli)),
			carapace.ActionFiles().Tag("server configuration"),
		).ToA(),
	)

	teamCmd.AddCommand(importCmd)

	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "Display a table of teamserver users and their status",
	}

	teamCmd.AddCommand(usersCmd)

	return teamCmd
}

// teamserversCompleter completes file paths to other teamserver application configs
func teamserversCompleter(cli *client.Client) carapace.CompletionCallback {
	return func(ctx carapace.Context) carapace.Action {
		var compErrors []carapace.Action
		homeDir, err := os.UserHomeDir()
		if err != nil {
			compErrors = append(compErrors, carapace.ActionMessage("failed to get user home dir: %w", err))
		}

		dirs, err := os.ReadDir(homeDir)
		if err != nil {
			compErrors = append(compErrors, carapace.ActionMessage("failed to list user directories: %w", err))
		}

		var results []string

		for _, dir := range dirs {
			if !strings.HasPrefix(dir.Name(), ".") {
				continue
			}
			if !dir.IsDir() {
				continue
			}
			if strings.TrimPrefix(dir.Name(), ".") != cli.Name() {
				continue
			}

			configPath := filepath.Join(homeDir, dir.Name(), "teamserver/client/configs")

			if configs, err := os.Stat(configPath); err == nil {
				if !configs.IsDir() {
					continue
				}

				files, _ := os.ReadDir(configPath)
				for _, file := range files {
					if !strings.HasSuffix(file.Name(), ".cfg") {
						continue
					}

					filePath := filepath.Join(configPath, file.Name())

					cfg, err := cli.ReadConfig(filePath)
					if err != nil || cfg == nil {
						continue
					}

					results = append(results, filePath)
					results = append(results, fmt.Sprintf("[%s] %s:%d", cfg.User, cfg.Host, cfg.Port))
				}
			}
		}

		return carapace.ActionValuesDescribed(results...).StyleF(style.ForPathExt).Tag("teamserver applications")
	}
}

func isNoConnect(cmd *cobra.Command) bool {
	noConnectCmds := []string{
		"import",
		"__complete",
		"_carapace",
	}

	for _, name := range noConnectCmds {
		if name == cmd.Name() {
			return true
		}
	}

	return false
}
