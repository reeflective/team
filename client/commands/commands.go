package commands

import (
	"encoding/json"
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

// Generate initliazes and returns a command tree to embed in client applications
// connecting to a teamserver. It requires only the client to use its functions.
func Generate(cli *client.Client) *cobra.Command {
	clientCmds := clientCommands(cli)
	return clientCmds
}

// PreRun returns a cobra command runner which connects the client to its teamserver.
// If the client is connected, nothing happens and its current connection reused, which
// makes this runner able to be ran in closed-loop consoles.
func PreRun(teamclient *client.Client) command.CobraRunnerE {
	return func(cmd *cobra.Command, args []string) error {
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cli.Connect(); err != nil {
				return err
			}

			// Server
			serverVer, err := cli.ServerVersion()
			if err != nil {
				fmt.Printf(command.Warn+"Server error: %s\n", err)
			}

			dirty := ""
			if serverVer.Dirty {
				dirty = fmt.Sprintf(" - %sDirty%s", command.Bold, command.Normal)
			}
			serverSemVer := fmt.Sprintf("%d.%d.%d", serverVer.Major, serverVer.Minor, serverVer.Patch)
			fmt.Printf(command.Info+"Server v%s - %s%s\n", serverSemVer, serverVer.Commit, dirty)

			// Client
			fmt.Printf(command.Info+"Client %s\n", version.Full())

			return nil
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
					if jsonErr, ok := err.(*json.SyntaxError); ok {
						fmt.Printf(command.Warn+"%s\n", jsonErr.Error())
					} else if err != nil {
						fmt.Printf(command.Warn+"%s\n", err.Error())
						continue
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
