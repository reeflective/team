package main

import (
	"log"

	serverCmd "github.com/reeflective/team/command/server"
	"github.com/reeflective/team/server"
	"github.com/rsteube/carapace"
)

func main() {
	// SERVER
	// Initialize the server
	serv := server.New("teamserver", server.WithDefaultPort(31340))

	// conn, _, err := serv.ServeLocal()
	// if err != nil {
	// 	log.Fatalf("Failed to serve: %s", err)
	// }
	//
	// defer serv.GracefulStop()
	//
	// // rpc.RegisterService(server)
	// // rpc.RegisterOtherService(server)
	//
	// // CLIENT
	// // And serve this to ourselves, to emulate a remote client.
	// client := client.New("example", client.WithConnection(conn))
	//
	// err = client.Connect()
	// if err != nil {
	// 	log.Printf("Failed to connect client: %s", err)
	// }
	//
	// if client.Conn() == nil {
	// 	log.Fatal("No client gRPC conn")
	// }
	// // rpc.RegisterServiceClient(conn)
	// // rpc.RegisterOtherServiceClient(conn)
	//
	// defer client.Disconnect()

	// fmt.Println(client.ServerVersion())

	// signs, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	// <-signs.Done()

	root := serverCmd.Commands(serv)

	carapace.Gen(root)
	err := root.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
