package main

import (
	"log"

	"github.com/reeflective/team/client"
	cli "github.com/reeflective/team/command/client"
	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
)

func main() {
	client := client.New("teamserver")

	rootCmd := cli.Commands(client)

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		err := client.Connect()
		if err != nil {
			log.Fatalf("Error connecting to teamserver: %s", err)
		}
		return nil
	}

	carapace.Gen(rootCmd)

	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
