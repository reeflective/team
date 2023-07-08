package command

import "github.com/spf13/cobra"

type (
	CobraRunnerE func(*cobra.Command, []string) error
	CobraRunner  func(*cobra.Command, []string)
)

const (
	TeamServerGroup     = "teamserver control" // TeamServerGroup is the group of all server/client control commands.
	UserManagementGroup = "user management"    // UserManagementGroup is the group to manage teamserver users.
)

// Colors // effects

const (
	// ANSI Colors
	Normal    = "\033[0m"
	Black     = "\033[30m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Orange    = "\033[33m"
	Blue      = "\033[34m"
	Purple    = "\033[35m"
	Cyan      = "\033[36m"
	Gray      = "\033[37m"
	Bold      = "\033[1m"
	Clearln   = "\r\x1b[2K"
	UpN       = "\033[%dA"
	DownN     = "\033[%dB"
	Underline = "\033[4m"

	// info - Display colorful information
	Info = Cyan + "[*] " + Normal
	// warn - warn a user
	Warn = Red + "[!] " + Normal
	// debugl - Display debugl information
	Debugl = Purple + "[-] " + Normal
)
