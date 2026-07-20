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

// Package boot resolves how a team application binary should run — as a thin
// CLIENT of a remote teamserver, or as an embedded SERVER — and dispatches to
// the matching setup.
//
// A single binary that embeds a teamserver is also, by construction, a client
// of itself; but when an operator has been handed a connection config (a
// "system" client config, written by `teamserver user --system`), the same
// binary should behave as a pure client of the REMOTE server and must never
// build a teamserver or open its database.
//
// boot.Run makes that split the default way to structure such a binary: the
// teamserver (and therefore its database, listeners and server-side filesystem)
// can only ever be constructed inside the Server callback, which Run invokes
// solely in server mode. Mode resolution itself has no server/database side
// effects.
package boot

import (
	"errors"

	"github.com/reeflective/team/client"
)

// Mode is a resolved run mode for a team application binary.
type Mode int

const (
	// ModeClient means a remote teamserver config was resolved (an explicit
	// one, or the current user's system config): the binary should run as a
	// thin client and must NOT build a teamserver or touch a database.
	ModeClient Mode = iota

	// ModeServer means no client config applies (or the server was forced):
	// the binary embeds and serves its own teamserver.
	ModeServer
)

// Boot describes how an application builds each of its two run modes. It is
// passed to Run, which resolves the mode once and invokes only the relevant
// callback — so the server (and the database/listeners it creates) never runs
// in client mode.
type Boot struct {
	// App is the application name, used to discover the current user's system
	// client config. Required.
	App string

	// ForceServer skips client-config resolution and runs in server mode. Set
	// it when the invocation is inherently server-side — for example when the
	// program was called as `<app> teamserver ...` (daemon/listen/user), which
	// must manage the local server even if a client config exists.
	ForceServer bool

	// Config, when non-nil, pins the remote teamserver config to run as a
	// client of (highest precedence, above system-config discovery). Use it to
	// honor an explicit --config flag or environment variable. Ignored when
	// ForceServer is set.
	Config *client.Config

	// Client runs the application in client mode. cfg is the resolved remote
	// server config (the pinned Config, or the discovered system config). It
	// MUST NOT construct a teamserver or open a database. Required.
	Client func(cfg *client.Config) error

	// Server runs the application in server mode. It is the ONLY place the
	// teamserver — and thus the database, listeners and server-side filesystem
	// — may be constructed. Required.
	Server func() error
}

// Resolve reports which mode Run would dispatch to, without invoking either
// callback and without any server/database side effects. It is exposed for
// callers that want to log or branch on the decision themselves; Run uses the
// same logic. A non-nil returned config accompanies ModeClient.
func (b Boot) Resolve() (Mode, *client.Config) {
	if b.ForceServer {
		return ModeServer, nil
	}

	if b.Config != nil {
		return ModeClient, b.Config
	}

	// Probe for a system client config WITHOUT server/database side effects:
	// an in-memory teamclient reads no logs/DB, yet SystemConfig still consults
	// the on-disk configs directory.
	probe, err := client.New(b.App, client.WithInMemory())
	if err != nil {
		return ModeServer, nil
	}

	if cfg, ok := probe.SystemConfig(); ok {
		return ModeClient, cfg
	}

	return ModeServer, nil
}

// Run resolves the application run mode and invokes the matching callback,
// guaranteeing the Server callback (and any database/listeners it builds) is
// never reached in client mode. It returns whatever the invoked callback
// returns, or an error if the Boot is missing a required field.
func Run(b Boot) error {
	if b.App == "" {
		return errors.New("boot: App is required")
	}

	if b.Client == nil {
		return errors.New("boot: Client callback is required")
	}

	if b.Server == nil {
		return errors.New("boot: Server callback is required")
	}

	switch mode, cfg := b.Resolve(); mode {
	case ModeClient:
		return b.Client(cfg)
	default:
		return b.Server()
	}
}
