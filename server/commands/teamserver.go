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
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/reeflective/team/internal/command"
	"github.com/reeflective/team/internal/systemd"
	"github.com/reeflective/team/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func daemoncmd(serv *server.Server) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}

		lhost, err := cmd.Flags().GetString("host")
		if err != nil {
			return fmt.Errorf("Failed to get --host flag: %s", err)
		}
		lport, err := cmd.Flags().GetUint16("port")
		if err != nil {
			return fmt.Errorf("Failed to get --port (%d) flag: %s", lport, err)
		}

		// Also written to logs in the teamserver code.
		defer func() {
			if r := recover(); r != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "stacktrace from panic: \n"+string(debug.Stack()))
			}
		}()

		// Blocking call, your program will only exit/resume on Ctrl-C/SIGTERM
		return serv.ServeDaemon(lhost, lport)
	}
}

func startListenerCmd(serv *server.Server) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}

		lhost, _ := cmd.Flags().GetString("host")
		lport, _ := cmd.Flags().GetUint16("port")
		persistent, _ := cmd.Flags().GetBool("persistent")
		ltype, _ := cmd.Flags().GetString("listener")

		_, err := serv.ServeAddr(ltype, lhost, lport)
		if err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), command.Info+"Teamserver listener started on %s:%d\n", lhost, lport)
			if persistent {
				serv.AddListener("", lhost, lport)
			}
		} else {
			return fmt.Errorf(command.Warn+"Failed to start job %v\n", err)
		}

		return nil
	}
}

func closeCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}

		for _, arg := range args {
			if arg == "" {
				continue
			}

			for _, ln := range serv.Listeners() {
				if strings.HasPrefix(ln.ID, arg) {
					err := serv.CloseListener(arg)
					if err != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), command.Warn, err)
					} else {
						fmt.Fprintf(cmd.OutOrStdout(), command.Info+"Closed %s listener (%d) [%s]", ln.Name, formatSmallID(ln.ID), ln.Description)
					}
				}
			}
		}
	}
}

func systemdConfigCmd(serv *server.Server) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}

		config := systemd.NewDefaultConfig()

		userf, _ := cmd.Flags().GetString("user")
		if userf != "" {
			config.User = userf
		}

		binPath, _ := cmd.Flags().GetString("binpath")
		if binPath != "" {
			config.Binpath = binPath
		}

		host, hErr := cmd.Flags().GetString("host")
		if hErr != nil {
			return hErr
		}

		port, pErr := cmd.Flags().GetUint16("port")
		if pErr != nil {
			return pErr
		}

		// The last argument is the systemd command:
		// its parent is the teamserver one, to which
		// should be attached the daemon command.
		daemonCmd, _, err := cmd.Parent().Find([]string{"daemon"})
		if err != nil {
			return fmt.Errorf("Failed to find teamserver daemon command in tree: %s", err)
		}

		config.Args = append(callerArgs(cmd.Parent()), daemonCmd.Name())
		if len(config.Args) > 0 && binPath != "" {
			config.Args[0] = binPath
		}

		if host != "" {
			config.Args = append(config.Args, strings.Join([]string{"--host", host}, " "))
		}

		if port != 0 {
			config.Args = append(config.Args, strings.Join([]string{"--port", strconv.Itoa(int(port))}, " "))
		}

		systemdConfig := systemd.NewFrom(serv.Name(), config)
		fmt.Fprintf(cmd.OutOrStdout(), systemdConfig)

		return nil
	}
}

func statusCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		if cmd.Flags().Changed("verbosity") {
			logLevel, err := cmd.Flags().GetCount("verbosity")
			if err == nil {
				serv.SetLogLevel(logLevel + int(logrus.ErrorLevel))
			}
		}

		// General options, available listeners, etc
		fmt.Fprintln(cmd.OutOrStdout(), formatSection("General"))

		// Logging files/level/status
		fmt.Fprintln(cmd.OutOrStdout(), formatSection("Logging"))

		// Listeners (excluding in-memory ones, BUT INCLUDING PERSISTENT NON-RUNNING ONES)
		fmt.Fprintln(cmd.OutOrStdout(), formatSection("Listeners"))

		listeners := serv.Listeners()
		cfg := serv.GetConfig()

		tb := &table.Table{}
		tb.SetStyle(teamserverTableStyle)

		tb.AppendHeader(table.Row{
			"ID",
			"Name",
			"Description",
			"State",
			"Persistent",
		})

		for _, ln := range listeners {
			persist := false
			for _, saved := range cfg.Listeners {
				if saved.ID == ln.ID {
					persist = true
				}
			}

			tb.AppendRow(table.Row{
				formatSmallID(ln.ID),
				ln.Name,
				ln.Description,
				command.Green + command.Bold + "Up" + command.Normal,
				persist,
			})
		}

		if len(listeners) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), tb.Render())
		}
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

var teamserverTableStyle = table.Style{
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

func formatSection(msg string, args ...any) string {
	return "\n" + command.Bold + command.Orange + fmt.Sprintf(msg, args...) + command.Normal
}

// formatSmallID returns a smallened ID for table/completion display.
func formatSmallID(id string) string {
	if len(id) <= 8 {
		return id
	}

	return id[:8]
}
