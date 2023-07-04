package main

import (
	"log"

	"github.com/reeflective/team/client"
	cli "github.com/reeflective/team/command/client"
	"github.com/rsteube/carapace"
)

func main() {
	client := client.New("teamserver")

	err := client.Connect()
	if err != nil {
		log.Fatalf("Error connecting to teamserver: %s", err)
	}

	rootCmd := cli.Commands(client)

	carapace.Gen(rootCmd)

	err = rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
