package main

import (
	"log"

	"github.com/rsteube/carapace"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/client/commands"
	grpc "github.com/reeflective/team/example/transports/grpc/client"
)

// main shows how to use a remote teamclient with a gRPC backend (transport & RPC).
func main() {
	// 1) Teamclient & dialers.
	//
	// We use the client grpc package to create a new teamclient dialer/RPC stack.
	// This client uses Mutual TLS authentication by default, and is thus to be used
	// as a client counterpart of the transport/grpc/server.Teamserver type.
	// This client also includes middleware logging.
	//
	// Note here that our gRPC client is a concrete type which implements
	// two different interfaces at once, so that our teamclient core is in
	// effect only-gRPC configured/enabled.
	gTeamclient := grpc.NewTeamClient()

	// Create a new teamclient core, specifying the transport/RPC backend to use.
	// The gRPC dialer also implements team.Client (Users/VersionServer), so passing
	// it with WithDialer() is enough: the core uses it as its backend automatically,
	// and there is no way to reach a remote server without at least such a dialer.
	teamclient, err := client.New("teamserver", client.WithDialer(gTeamclient))
	if err != nil {
		log.Fatal(err)
	}

	// 2) Commands
	//
	// Since this binary is a teamclient-only application, we only need
	// the teamclient command tree, to query users, server version and
	// to manage/import remote teamserver config files.
	// Like for teamserver commands, this tree will use teamclient connection
	// functions and utilities to connect/disconnect when required to.
	root := commands.Generate(teamclient)

	// Completions
	carapace.Gen(root)

	// Execute one of the teamserver commands.
	err = root.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
