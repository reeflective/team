package main

import (
	"log"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/client/commands"
	grpc "github.com/reeflective/team/transports/grpc/client"
	"github.com/rsteube/carapace"
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
	// There is no way to make a teamclient work with a remote server if it is not
	// being given at least a team.Client type, and will most likely involve passing
	// a full dialer backend like in the call below.
	teamclient, err := client.New("teamserver", gTeamclient, client.WithDialer(gTeamclient))
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

func mainPreCommands() {}
