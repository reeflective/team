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

// Package log provides the reeflective/team console logging system, so that
// applications embedding a teamclient/teamserver can reuse and restyle the exact
// same logging their team core uses.
//
// The heart is ConsoleHandler, an slog.Handler rendering aligned, colored lines
// of the form "<time> <LEVEL> <package> <message>", with every column (level
// markers and colors, package/time/message colors, column widths, timestamp)
// configurable through ConsoleOptions and LevelStyle.
//
// Typical uses:
//   - Log in your own code with the team console style: NewConsole(opts), then
//     Named(logger, pkg, stream) to get the aligned package column.
//   - Replace the team core's logging entirely: pass a handler to
//     server.WithLogger / client.WithLogger.
//   - Only restyle the built-in console while keeping the core's file logger:
//     server.WithConsoleOptions / client.WithConsoleOptions.
//
// Because the core's filesystem is an afero.Afero (returned by
// Server.Filesystem() / Client.Filesystem()), a consumer running fully in-memory
// can build a Logger with New against that same ephemeral filesystem.
package log
