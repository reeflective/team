package main

import (
	"log"

	teamclient "github.com/reeflective/team/client"
	"github.com/reeflective/team/client/commands"
	grpc "github.com/reeflective/team/transports/grpc/client"
	"github.com/rsteube/carapace"
)

func main() {
	client, dialer := grpc.NewTeamClient()

	// Create a new teamserver client, without any working
	// gRPC connection at this stage. We could pass some options
	// to it if we want to customize behavior.
	teamclient, err := teamclient.New("teamserver", client, teamclient.WithDialer(dialer), teamclient.WithInMemory())
	if err != nil {
		log.Fatal(err)
	}

	// Let the teamserver client dedicated command tree make use of it.
	root := commands.Generate(teamclient)

	// We are responsible for connecting the client, however this
	// is on purpose: there are various cases where you don't want
	// to connect unconditionally.
	// Here, for example, all commands of our application have a single-exec mode,
	// and we need to ensure both connection before, and graceful disconnect after.
	// Another example, in a closed console, would not involve disonnect post-runs.

	// Completions
	carapace.Gen(root)

	// Run your application: anything having to do with
	// the teamserver or one of its commands, will be done
	err = root.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
