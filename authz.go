package team

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

// Authorizer decides whether an authenticated user may perform an action.
//
// The team library performs authentication (resolving an identity/[User]);
// authorization is the application's responsibility. Implement this interface
// to plug an authorization policy into team-based components.
//
// Authorizer is deliberately transport-agnostic. The same Authorizer can gate a
// transport interceptor (e.g. a gRPC middleware), a background/detached task, an
// event handler, or any other in-process path to a privileged capability —
// including code paths where the original request context is no longer
// available. This lets an application enforce a single authorization rule source
// regardless of *how* a capability is reached, closing the common and dangerous
// gap where authorization is bolted onto one transport and silently bypassed by
// another code path that calls the same capability directly.
type Authorizer interface {
	// Authorize reports whether the named (already-authenticated) user may
	// perform action. A nil error means the action is allowed; a non-nil error
	// denies it and should explain why.
	//
	// action is an application-defined capability identifier (for example an RPC
	// method name, or a namespaced capability such as "ai:call-tool"). Both
	// arguments are values the caller can supply without a live request context,
	// so an Authorizer can be consulted from anywhere in the process.
	Authorize(user string, action string) error
}
