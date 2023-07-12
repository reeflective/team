package main

import (
	"log"

	teamclient "github.com/reeflective/team/client"
	teamserver "github.com/reeflective/team/server"
	"github.com/reeflective/team/server/commands"
	grpc "github.com/reeflective/team/transports/grpc/server"
	"github.com/rsteube/carapace"
)

func main() {
	// Create a teamserver.
	// This server can handle any number of remote clients for a given application
	// named "teamserver", including its own local runtime (fully in-memory) client.
	//
	// This call to create the server only creates the application default directory.
	// No files, logs, connections or any interaction with the os/filesystem are made.
	// TODO: Rewrite comments
	listener := grpc.NewTeamServer()

	teamserver, err := teamserver.New("teamserver", listener)
	if err != nil {
		log.Fatal(err)
	}

	teamserver.GetUsers()

	// TODO: write comments
	client, dialer := grpc.DialerFrom(listener)

	teamclient, err := teamclient.New(teamserver.Name(), client, teamclient.WithDialer(dialer))
	if err != nil {
		log.Fatal(err)
	}

	// Generate a tree of server-side commands: this tree also has client-only
	// commands as a subcommand "client" of the "teamserver" command root here.
	serverCmds := commands.Generate(teamserver, teamclient)

	// Generate completions for the tree.
	carapace.Gen(serverCmds)

	// Run our teamserver binary.
	// Here we are executing the simple teamserver command tree, but anything is possible:
	// - We could "forget" about our teamserver because we have a blocking call somewhere,
	// - Let one of the commands to be executed and exit, without listening anywhere.
	// - Many, many different variants in which you can keep working below.
	err = serverCmds.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func mainInMemory() {
	var clientOpts []teamclient.Options
	clientOpts = append(clientOpts,
		teamclient.WithInMemory(),
	)

	var serverOpts []teamclient.Options
	serverOpts = append(serverOpts,
		teamclient.WithInMemory(),
	)
}

func mainIntegrated() {}

func mainSmallest() {}

func mainCustom() {}

func mainNoCommands() {}
