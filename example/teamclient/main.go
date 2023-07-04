package main

import (
	"log"

	"github.com/reeflective/team/client"
	cli "github.com/reeflective/team/command/client"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func main() {
	// Create a new teamserver client, without any working
	// gRPC connection at this stage. We could pass some options
	// to it if we want to customize behavior.
	client := client.New("teamserver")

	// Let the teamserver client dedicated command tree make use of it.
	rootCmd := cli.Commands(client)

	// Only connect to the server before actually running commands.
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		err := client.Connect()
		if err != nil {
			log.Fatalf("Error connecting to teamserver: %s", err)
		}
		return nil
	}

	// Completions
	carapace.Gen(rootCmd)

	// Run your application: anything having to do with
	// the teamserver or one of its commands, will be done
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
