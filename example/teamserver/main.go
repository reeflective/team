package main

import (
	"log"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"

	"github.com/reeflective/team/client"
	cli "github.com/reeflective/team/command/client"
	serverCmd "github.com/reeflective/team/command/server"
	"github.com/reeflective/team/server"
)

func main() {
	//
	// 1) Server
	//
	serv := server.New("teamserver", server.WithDefaultPort(31340))

	// We want to be able both to use ourself as a client
	// (transparently interfacing with a remote/local server),
	// without caring about networking in the first place.
	//
	// We generate a local client (and its command tree) from our
	// server, and bind these client commands to our server tree.
	conn, _, err := serv.ServeLocal()
	if err != nil {
		log.Fatalf("Failed to serve: %s", err)
	}
	defer serv.GracefulStop()

	// Generate the tree of server-side commands, using this server.
	root := serverCmd.Commands(serv)

	//
	// 2) Client
	//

	// We create a teamclient counterpart, but specifying we already
	// have a physical gRPC connection to use over any remote ones.
	// And we serve this conn to oursef, to emulate a remote client.
	client := client.New("teamserver", client.WithConnection(conn))

	// Generate the tree of the client-side commands. and its root as
	// a sister of the server ones.
	// This has the advantage of containing all teamserver client/server
	// commands under a single root (to bind wherever you want in another
	// command tree). Also, server/client trees are still clearly delimited.
	clientCmds := cli.Commands(client)
	clientCmds.Use = "client"
	root.AddCommand(clientCmds)

	// Only connect to the server before actually running commands.
	// Those runners are not bound by default, since users might
	// want to use them differently.
	clientCmds.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		err := client.Connect()
		if err != nil {
			log.Fatalf("Error connecting to teamserver: %s", err)
		}
		return nil
	}

	clientCmds.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		client.Disconnect()
	}

	// Completions
	carapace.Gen(root)

	// Run our binary. Here we are executing the simple teamserver command tree,
	// but anything is possible, really:
	// - "forgetting" about your teamserver because you have a blocking call somewhere,
	// - Let one of the commands to be executed and exit, without listening anywhere.
	// - Many, many different variants in which you can keep working below.
	err = root.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
