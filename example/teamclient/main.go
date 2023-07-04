package main

import (
	"log"

	"github.com/reeflective/team/client"
	cli "github.com/reeflective/team/command/client"
	"github.com/rsteube/carapace"
)

func main() {
	// Create a new teamserver client, without any working
	// gRPC connection at this stage. We could pass some options
	// to it if we want to customize behavior.
	client := client.New("teamserver")

	// Let the teamserver client dedicated command tree make use of it.
	root := cli.Commands(client)

	// We are responsible for connecting the client, however this
	// is on purpose: there are various cases where you don't want
	// to connect unconditionally.
	// Here, for example, all commands of our application have a single-exec mode,
	// and we need to ensure both connection before, and graceful disconnect after.
	// Another example, in a closed console, would not involve disonnect post-runs.
	for _, cmd := range root.Commands() {
		cmd.PersistentPreRunE = cli.ConnectRun(client)
		cmd.PersistentPostRunE = cli.DisconnectRun(client)
	}

	// Completions
	carapace.Gen(root)

	// Run your application: anything having to do with
	// the teamserver or one of its commands, will be done
	err := root.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
