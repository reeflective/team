package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/reeflective/team/command/client"
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
		&cobra.Group{ID: client.TeamServerGroup, Title: client.TeamServerGroup},
		&cobra.Group{ID: client.UserManagementGroup, Title: client.UserManagementGroup},
	)

	// [ Listeners and servers control commands ] ------------------------------------------

	listenCmd := &cobra.Command{
		Use:     "listen",
		Short:   "Start a teamserver gRPC listener job (non-blocking)",
		GroupID: client.TeamServerGroup,
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
		GroupID: client.TeamServerGroup,
		Run:     daemoncmd(server),
	}
	daemonCmd.Flags().StringP("host", "l", "-", "multiplayer listener host")
	daemonCmd.Flags().Uint16P("port", "p", uint16(0), "multiplayer listener port")

	teamCmd.AddCommand(daemonCmd)

	systemdCmd := &cobra.Command{
		Use:     "systemd",
		Short:   "Print a systemd unit file for the application teamserver, with options",
		GroupID: client.TeamServerGroup,
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
		GroupID: client.UserManagementGroup,
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

	// TODO:: Find other directories in home that are likely other applications,
	// and parse them for completions descriptions, keeping their credentials but changing host/port in the copy.

	// Delete and kick user
	rmUserCmd := &cobra.Command{
		Use:     "delete",
		Short:   "Remove a user from the teamserver, and revoke all its current tokens",
		GroupID: client.UserManagementGroup,
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
		GroupID: client.UserManagementGroup,
		Args:    cobra.ExactArgs(1),
		Run:     importCACmd(server),
	}

	carapace.Gen(cmdImportCA).PositionalCompletion(carapace.ActionFiles())
	teamCmd.AddCommand(cmdImportCA)

	// Export the list of users and their credentials.
	cmdExportCA := &cobra.Command{
		Use:     "export",
		Short:   "Export a Certificate Authority file containing the teamserver users",
		GroupID: client.UserManagementGroup,
		Args:    cobra.RangeArgs(0, 1),
		Run:     exportCACmd(server),
	}

	carapace.Gen(cmdExportCA).PositionalCompletion(carapace.ActionFiles())
	teamCmd.AddCommand(cmdExportCA)

	return teamCmd
}

func daemoncmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		lhost, err := cmd.Flags().GetString("host")
		if err != nil {
			fmt.Printf("Failed to parse --host flag %s\n", err)
			return
		}
		lport, err := cmd.Flags().GetUint16("port")
		if err != nil {
			fmt.Printf("Failed to parse --port flag %s\n", lport, err)
			return
		}

		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic:\n%s", debug.Stack())
				fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
				os.Exit(99)
			}
		}()

		serv.ServeDaemon(lhost, lport)
	}
}

func startListenerCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		lhost, _ := cmd.Flags().GetString("host")
		lport, _ := cmd.Flags().GetUint16("port")
		persistent, _ := cmd.Flags().GetBool("persistent")

		_, _, err := serv.ServeAddr(lhost, lport)
		if err == nil {
			fmt.Printf(info+"Teamserver listener started on %s:%d\n", lhost, lport)
			if persistent {
				serv.AddListenerJob(&server.ListenerConfig{
					Host: lhost,
					Port: lport,
				})
			}
		} else {
			fmt.Printf(warn+"Failed to start job %v\n", err)
		}
	}
}

func systemdConfigCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		config := server.DefaultSystemdConfig()

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

		systemdConfig := serv.GenerateServiceFile(config)
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

func createUserCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		name, _ := cmd.Flags().GetString("name")
		lhost, _ := cmd.Flags().GetString("host")
		lport, _ := cmd.Flags().GetUint16("port")
		save, _ := cmd.Flags().GetString("save")
		system, _ := cmd.Flags().GetBool("system")

		if save == "" {
			save, _ = os.Getwd()
		}

		var filename string
		var saveTo string

		if system {
			user, err := user.Current()
			if err != nil {
				fmt.Printf(warn, "Failed to get current OS user: %", err)
				return
			}
			name = user.Username
			filename = fmt.Sprintf("%s_%s_default", serv.Name(), user.Username)
			saveTo = serv.ClientConfigsDir()
		} else {
			saveTo, _ = filepath.Abs(save)
			fi, err := os.Stat(saveTo)
			if !os.IsNotExist(err) && !fi.IsDir() {
				fmt.Printf(warn+"File already exists %s\n", err)
				return
			}

			if !os.IsNotExist(err) && fi.IsDir() {
				filename = fmt.Sprintf("%s_%s", filepath.Base(name), filepath.Base(lhost))
			}
		}

		fmt.Printf(info + "Generating new client certificate, please wait ... \n")
		configJSON, err := serv.NewUserConfig(name, lhost, lport)
		if err != nil {
			fmt.Printf(warn+"%s\n", err)
			return
		}

		saveTo = filepath.Join(saveTo, filename+".cfg")
		err = ioutil.WriteFile(saveTo, configJSON, 0o600)
		if err != nil {
			fmt.Printf(warn+"Failed to write config to %s (%s) \n", saveTo, err)
			return
		}

		fmt.Printf(info+"Saved new client config to: %s \n", saveTo)
	}
}

func rmUserCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		operator := args[0]

		fmt.Printf(info+"Removing client certificate(s)/token(s) for %s, please wait ... \n", operator)

		err := serv.DeleteUser(operator)
		if err != nil {
			fmt.Printf(warn+"Failed to remove the user certificate: %v \n", err)
			return
		}

		fmt.Printf(info+"User %s has been deleted from the teamserver, and kicked out.\n", operator)
	}
}

func importCACmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		load := args[0]

		fi, err := os.Stat(load)
		if os.IsNotExist(err) || fi.IsDir() {
			fmt.Printf("Cannot load file %s\n", load)
			os.Exit(1)
		}

		data, err := os.ReadFile(load)
		if err != nil {
			fmt.Printf("Cannot read file %s", err)
			os.Exit(1)
		}

		// CA - Exported CA format
		type CA struct {
			Certificate string `json:"certificate"`
			PrivateKey  string `json:"private_key"`
		}

		importCA := &CA{}
		err = json.Unmarshal(data, importCA)
		if err != nil {
			fmt.Printf("Failed to parse file %s", err)
			os.Exit(1)
		}
		cert := []byte(importCA.Certificate)
		key := []byte(importCA.PrivateKey)
		serv.SaveUsersCA(cert, key)
	}
}

func exportCACmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		var save string
		if len(args) == 1 {
			save = args[0]
		}

		if strings.TrimSpace(save) == "" {
			save, _ = os.Getwd()
		}

		certificateData, privateKeyData, err := serv.GetUsersCA()
		if err != nil {
			fmt.Printf("Error reading CA %s\n", err)
			os.Exit(1)
		}

		// CA - Exported CA format
		type CA struct {
			Certificate string `json:"certificate"`
			PrivateKey  string `json:"private_key"`
		}

		exportedCA := &CA{
			Certificate: string(certificateData),
			PrivateKey:  string(privateKeyData),
		}

		saveTo, _ := filepath.Abs(save)
		fi, err := os.Stat(saveTo)
		if !os.IsNotExist(err) && !fi.IsDir() {
			fmt.Printf("File already exists: %s\n", err)
			os.Exit(1)
		}
		if !os.IsNotExist(err) && fi.IsDir() {
			filename := fmt.Sprintf("%s.ca", filepath.Base("user"))
			saveTo = filepath.Join(saveTo, filename)
		}
		data, _ := json.Marshal(exportedCA)
		err = os.WriteFile(saveTo, data, 0o600)
		if err != nil {
			fmt.Printf("Write failed: %s (%s)\n", saveTo, err)
			os.Exit(1)
		}
	}
}
