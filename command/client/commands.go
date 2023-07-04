package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/rsteube/carapace/pkg/style"
	"github.com/spf13/cobra"

	"github.com/reeflective/team/client"
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

const (
	TeamServerGroup     = "teamserver control" // TeamServerGroup is the group of all server/client control commands.
	UserManagementGroup = "user management"    // UserManagementGroup is the group to manage teamserver users.
)

// Commands initliazes and returns a command tree to embed in client applications
// connecting to a teamserver. It requires the client itself to use its functions.
func Commands(cli *client.Client) *cobra.Command {
	teamCmd := &cobra.Command{
		Use:     "teamclient",
		Short:   "Client-only teamserver commands (import configs, show users, etc)",
		GroupID: TeamServerGroup,
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print teamserver client version",
		Run: func(cmd *cobra.Command, args []string) {
			// Server first
			serverVer, err := cli.ServerVersion()
			if err != nil {
				fmt.Printf(warn+"Server (error: %s)\r\n", err)
			}
			dirty := ""
			if serverVer.Dirty {
				dirty = fmt.Sprintf(" - %sDirty%s", bold, normal)
			}
			serverSemVer := fmt.Sprintf("%d.%d.%d", serverVer.Major, serverVer.Minor, serverVer.Patch)
			fmt.Printf(info+"Server v%s - %s%s\n", serverSemVer, serverVer.Commit, dirty)

			// Client
			fmt.Printf(info+"Client %s\n", client.FullVersion())
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

// ConnectRun returns a cobra command connecting the client to the teamserver.
// This should generally be used as one of (or part of another) command pre-runner.
func ConnectRun(cli *client.Client, opts ...client.Options) func(cmd *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		if cli.IsConnected() {
			return nil
		}

		if err := cli.Connect(opts...); err != nil {
			fmt.Printf(warn+"Error connecting to teamserver: %s\n", err)
			return err
		}

		return nil
	}
}

// DisconnectRun returns a cobra command disconnecting the client from the teamserver.
// This should generally be used as one of (or part of another) command post-runner.
func DisconnectRun(cli *client.Client) func(cmd *cobra.Command, _ []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		// THis is safe, client ensures to close what can be.
		cli.Disconnect()
		return nil
	}
}

// teamserversCompleter completes file paths to other teamserver application configs
func teamserversCompleter(cli *client.Client) carapace.CompletionCallback {
	return func(ctx carapace.Context) carapace.Action {
		var compErrors []carapace.Action
		homeDir, err := os.UserHomeDir()
		if err != nil {
			compErrors = append(compErrors, carapace.ActionMessage("failed to get user home dir: %s", err))
		}

		dirs, err := os.ReadDir(homeDir)
		if err != nil {
			compErrors = append(compErrors, carapace.ActionMessage("failed to list user directories: %s", err))
		}

		var results []string

		for _, dir := range dirs {
			if !strings.HasPrefix(dir.Name(), ".") {
				continue
			}
			if !dir.IsDir() {
				continue
			}
			if strings.TrimPrefix(dir.Name(), ".") == cli.Name() {
				continue
			}

			configPath := filepath.Join(homeDir, dir.Name(), "configs")

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
				}
			}
		}

		return carapace.ActionValues(results...).StyleF(style.ForPathExt).Tag("teamserver applications")
	}
}
