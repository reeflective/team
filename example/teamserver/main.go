package main

import (
	"log"

	"github.com/reeflective/team/client"
	"github.com/reeflective/team/server"
	"github.com/reeflective/team/server/commands"
	grpc "github.com/reeflective/team/transports/grpc/server"
	"github.com/rsteube/carapace"
)

// main shows how to use a teamserver and teamclient with gRPC backends (transport & RPC).
func main() {
	// 1) Teamserver & listeners
	//
	// Create a new gTeamserver, implementing the team/server.Listener interface.
	// This gTeamserver is able to serve both remote (TLS-authenticated) teamclients, and
	// in-memory (non-authenticated/encrypted) ones alike. See below when creating the teamclient.
	//
	// We don't pass any specific server-side gRPC options, as this gTeamserver is already
	// configured with the correct MutualTLS(auth/enc) and logging middlewares.
	gTeamserver := grpc.NewListener()

	// Create a new application teamserver core.
	// This server can handle any number of remote clients for an application here
	// named "teamserver", including any number of runtime (in-memory) self-clients.
	//
	// This call only creates the app default private directories, and does not yet
	// write to them. It does not connect to its configured database backend yet.
	//
	// We register our gRPC listener for the teamserver to be able to serve remote teamclients.
	teamserver, err := server.New("teamserver", server.WithListener(gTeamserver))
	if err != nil {
		log.Fatal(err)
	}

	// 2) Teamclients & dialers
	//
	// Although the teamserver above can be a client of itself without further addings,
	// we want to use the same backend for both remote and local gRPC app teamclients.
	// We thus generate a local gTeamclient (almost identical to remote ones), which can
	// use the generic gRPC listener backend.
	gTeamclient := grpc.NewClientFrom(gTeamserver)

	// Create a new teamclient core, just like we would in a client-only binary application.
	// We specify which dialing backend we want (in this case, our custom gRPC dialer/RPC client)
	// Note here that our gRPC client is a concrete type which implements two different interfaces
	// at once, so that our teamclient core is in effect only-gRPC configured/enabled.
	teamclient, err := client.New(teamserver.Name(), gTeamclient, client.WithDialer(gTeamclient))
	if err != nil {
		log.Fatal(err)
	}

	// 3) Commands
	//
	// Our application is a simple teamserver we want to use and manage from the CLI.
	// Teamclients and teamservers each possess a specific tree of control/use commands.
	//
	// Here and as always, any teamserver can be a client of itself: thus the command tree
	// below also has its own teamclient command tree, bound as a subcommand "client" of the
	// "teamserver" command root here.
	serverCmds := commands.Generate(teamserver, teamclient)

	// Generate completions for the tree.
	// These commands are all equipped with various and powerful completions for
	// users listing, daemon & listeners control, configurations management, etc.
	carapace.Gen(serverCmds)

	// Execute one of the teamserver commands:
	// This could be simply creating a new user, saving/or deleting a new persistent listener,
	// showing the teamserver status, or even blocking and serve the teamserver as a daemon.
	err = serverCmds.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func mainSmallest() {
	teamserver, err := server.New("smallserver", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Generate a tree of server-side commands: this tree also has client-only
	// commands as a subcommand "client" of the "teamserver" command root here.
	serverCmds := commands.Generate(teamserver, teamserver.Self())
	serverCmds.Use = "smallserver"

	// Generate completions for the tree.
	carapace.Gen(serverCmds)

	// Run our teamserver binary.
	err = serverCmds.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

// Run our teamserver binary.
// Here we are executing the simple teamserver command tree, but anything is possible:
// // - We could "forget" about our teamserver because we have a blocking call somewhere,
// // - Let one of the commands to be executed and exit, without listening anywhere.
// // - Many, many different variants in which you can keep working below.

func mainInMemory() {
	var clientOpts []client.Options
	clientOpts = append(clientOpts,
		client.WithInMemory(),
	)

	var serverOpts []client.Options
	serverOpts = append(serverOpts,
		client.WithInMemory(),
	)
}

func mainIntegrated() {}

func mainCustom() {
}

func mainNoCommands() {}
