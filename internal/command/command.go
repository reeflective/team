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
