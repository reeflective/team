package client

import (
	"fmt"
	"os"

	"github.com/rsteube/carapace"
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

// Commands initliazes and returns a command tree to embed in client applications
// connecting to a teamserver. It requires the client itself to use its functions.
func Commands(cli *client.Client) *cobra.Command {
	teamCmd := &cobra.Command{
		Use:   "teamclient",
		Short: "Manage the client-side application of the teamserver, and users",
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

	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "Display a table of teamserver users and their status",
	}

	teamCmd.AddCommand(usersCmd)

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

	carapace.Gen(importCmd).PositionalCompletion(carapace.ActionFiles().Tag("server configuration"))

	teamCmd.AddCommand(importCmd)

	return teamCmd
}
