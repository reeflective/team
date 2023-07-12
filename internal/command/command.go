package command

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
)

type (
	CobraRunnerE func(*cobra.Command, []string) error
	CobraRunner  func(*cobra.Command, []string)
)

const (
	TeamServerGroup     = "teamserver control" // TeamServerGroup is the group of all server/client control commands.
	UserManagementGroup = "user management"    // UserManagementGroup is the group to manage teamserver users.
)

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

	// info - Display colorful information.
	Info = Cyan + "[*] " + Normal
	// warn - warn a user.
	Warn = Red + "[!] " + Normal
	// debugl - Display debugl information.
	Debugl = Purple + "[-] " + Normal
)

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
