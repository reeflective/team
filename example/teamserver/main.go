package main

import (
	"log"

	"github.com/rsteube/carapace"

	"github.com/reeflective/team/client"
	cli "github.com/reeflective/team/command/server"
	"github.com/reeflective/team/server"
)

func main() {
	// Create a teamserver and a teamclient.
	// None of those yet have a working RPC connection, and the server
	// is not yet connected to its database, loggers and certificates.
	teamServer := server.New("teamserver", server.WithDefaultPort(31340))
	teamClient := client.New("teamserver")

	// Pass both server and clients to the commands package:
	// we are being given two command trees: teamserver ones (server only)
	// and teamclient ones. Both are configured with pre-runners that will
	// connect themselves together over an in-memory gRPC connection.
	serverCmds, clientCmds := cli.ConnectLocal(teamServer, teamClient)

	// Add the teamclient command tree as a subtree of the server ones.
	// In this case, the teamserver is the application itself: it is not
	// part of a larger set of domain-specific commands, which would be
	// the case in normal use cases for this library.
	serverCmds.AddCommand(clientCmds)

	// Generate completions for the tree.
	carapace.Gen(serverCmds)

	// Run our teamserver binary.
	// Here we are executing the simple teamserver command tree, but anything is possible:
	// - We could "forget" about our teamserver because we have a blocking call somewhere,
	// - Let one of the commands to be executed and exit, without listening anywhere.
	// - Many, many different variants in which you can keep working below.
	err := serverCmds.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
