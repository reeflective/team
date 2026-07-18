package command

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
	"github.com/carapace-sh/carapace"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"github.com/reeflective/team/log"
)

type (
	// CobraRunnerE is a cobra runner returning an error.
	CobraRunnerE func(*cobra.Command, []string) error
	// CobraRunner is a cobra runner returning nothing.
	CobraRunner func(*cobra.Command, []string)
)

const (
	// ClientConfigExt is the client remote server config file extension.
	ClientConfigExt = "teamclient.cfg"
	// ServerConfigExt is the server backend config file extension.
	ServerConfigExt = "teamserver.json"
)

const (
	// TeamServerGroup is the group of all server/client control commands.
	TeamServerGroup = "teamserver control"
	// UserManagementGroup is the group to manage teamserver users.
	UserManagementGroup = "user management"
)

// LogFormatCompleter completes the --log-format flag values (console/text/json),
// each with a short description, for both teamserver and teamclient trees.
func LogFormatCompleter() carapace.Action {
	return carapace.ActionCallback(func(carapace.Context) carapace.Action {
		values := make([]string, 0, len(log.Formats())*2)
		for _, format := range log.Formats() {
			values = append(values, format.String(), format.Describe())
		}

		return carapace.ActionValuesDescribed(values...)
	})
}

// Colors / effects.
const (
	// ANSI Colors.
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

	// Info - Display colorful information.
	Info = Cyan + "[*] " + Normal
	// Warn - warn a user.
	Warn = Red + "[!] " + Normal
	// Debugl - Display debugl information.
	Debugl = Purple + "[-] " + Normal
)

// TableStyle is a default table style for users and listeners.
var TableStyle = table.Style{
	Name: "TeamServerDefault",
	Box: table.BoxStyle{
		BottomLeft:       " ",
		BottomRight:      " ",
		BottomSeparator:  " ",
		Left:             " ",
		LeftSeparator:    " ",
		MiddleHorizontal: "=",
		MiddleSeparator:  " ",
		MiddleVertical:   " ",
		PaddingLeft:      " ",
		PaddingRight:     " ",
		Right:            " ",
		RightSeparator:   " ",
		TopLeft:          " ",
		TopRight:         " ",
		TopSeparator:     " ",
		UnfinishedRow:    "~~",
	},
	Color: table.ColorOptions{
		IndexColumn:  text.Colors{},
		Footer:       text.Colors{},
		Header:       text.Colors{},
		Row:          text.Colors{},
		RowAlternate: text.Colors{},
	},
	Format: table.FormatOptions{
		Footer: text.FormatDefault,
		Header: text.FormatTitle,
		Row:    text.FormatDefault,
	},
	Options: table.Options{
		DrawBorder:      false,
		SeparateColumns: true,
		SeparateFooter:  false,
		SeparateHeader:  true,
		SeparateRows:    false,
	},
}
