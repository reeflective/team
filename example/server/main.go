package main

import (
	"log"

	"github.com/rsteube/carapace"

	teamclient "github.com/reeflective/team/client"
	cli "github.com/reeflective/team/command/server"
	teamserver "github.com/reeflective/team/server"
	"github.com/reeflective/team/server/transports/grpc"
)

func main() {
	// Create a teamserver.
	// This server can handle any number of remote clients for a given application
	// named "teamserver", including its own local runtime (fully in-memory) client.
	//
	// This call to create the server only creates the application default directory.
	// No files, logs, connections or any interaction with the os/filesystem are made.
	// TODO: Rewrite comments
	listener := grpc.NewServer()

	server, err := teamserver.New("teamserver", listener)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: write comments
	dialer := grpc.DialerFrom(listener)

	client, err := teamclient.New(server.Name(), dialer)
	if err != nil {
		log.Fatal(err)
	}

	// Generate a tree of server-side commands: this tree also has client-only
	// commands as a subcommand "client" of the "teamserver" command root here.
	serverCmds := cli.Commands(server, client)

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
