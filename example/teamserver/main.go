package main

import (
	"log"

	"github.com/reeflective/team/client"
	grpc "github.com/reeflective/team/example/transports/grpc/server"
	"github.com/reeflective/team/server"
	"github.com/reeflective/team/server/commands"
	"github.com/rsteube/carapace"
)

// mainSmallest is the smallest example of a teamserver usage.
// The latter can only serve itself in-memory, since there are no
// remote teamserver listener stacks registered with it. Still, the
// teamserver functionality is complete and works identically regardless.
func mainSmallest() {
	teamserver, err := server.New("smallserver")
	if err != nil {
		log.Fatal(err)
	}

	// Generate a tree of server-side commands: this tree also has client-only
	// commands as a subcommand "client" of the "teamserver" command root here.
	serverCmds := commands.Generate(teamserver, teamserver.Self())
	serverCmds.Use = "smallserver"

	// Generate completions for the tree.
	carapace.Gen(serverCmds)

	// Run our teamserver binary.
	err = serverCmds.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

// main shows how to use a teamserver and teamclient with gRPC backends (transport & RPC).
func main() {
	// 1) Teamserver & listeners
	//
	// Create a new gTeamserver, implementing the team/server.Listener interface.
	// This gTeamserver is able to serve both remote (TLS-authenticated) teamclients, and
	// in-memory (non-authenticated/encrypted) ones alike. See below when creating the teamclient.
	//
	// We don't pass any specific server-side gRPC options, as this gTeamserver is already
	// configured with the correct MutualTLS(auth/enc) and logging middlewares.
	gTeamserver := grpc.NewListener()

	// Create a new application teamserver core.
	// This server can handle any number of remote clients for an application here
	// named "teamserver", including any number of runtime (in-memory) self-clients.
	//
	// This call only creates the app default private directories, and does not yet
	// write to them. It does not connect to its configured database backend yet.
	//
	// We register our gRPC listener for the teamserver to be able to serve remote teamclients.
	teamserver, err := server.New("teamserver", server.WithListener(gTeamserver))
	if err != nil {
		log.Fatal(err)
	}

	// 2) Teamclients & dialers
	//
	// Although the teamserver above can be a client of itself without further addings,
	// we want to use the same backend for both remote and local gRPC app teamclients.
	// We thus generate a local gTeamclient (almost identical to remote ones), which can
	// use the generic gRPC listener backend.
	gTeamclient := grpc.NewClientFrom(gTeamserver)

	// Create a new teamclient core, just like we would in a client-only binary application.
	// We specify which dialing backend we want (in this case, our custom gRPC dialer/RPC client)
	// Note here that our gRPC client is a concrete type which implements two different interfaces
	// at once, so that our teamclient core is in effect only-gRPC configured/enabled.
	teamclient, err := client.New(teamserver.Name(), gTeamclient, client.WithDialer(gTeamclient))
	if err != nil {
		log.Fatal(err)
	}

	// 3) Commands
	//
	// Our application is a simple teamserver we want to use and manage from the CLI.
	// Teamclients and teamservers each possess a specific tree of control/use commands.
	//
	// Here and as always, any teamserver can be a client of itself: thus the command tree
	// below also has its own teamclient command tree, bound as a subcommand "client" of the
	// "teamserver" command root here.
	serverCmds := commands.Generate(teamserver, teamclient)

	// Generate completions for the tree.
	// These commands are all equipped with various and powerful completions for
	// users listing, daemon & listeners control, configurations management, etc.
	carapace.Gen(serverCmds)

	// Execute one of the teamserver commands:
	// This could be simply creating a new user, saving/or deleting a new persistent listener,
	// showing the teamserver status, or even blocking and serve the teamserver as a daemon.
	err = serverCmds.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

// mainSmallGRPC is the equivalent of main, without comments.
func mainSmallGRPC() {
	// Server
	gTeamserver := grpc.NewListener()

	teamserver, err := server.New("teamserver", server.WithListener(gTeamserver))
	if err != nil {
		log.Fatal(err)
	}

	// Client
	gTeamclient := grpc.NewClientFrom(gTeamserver)

	teamclient := teamserver.Self(client.WithDialer(gTeamclient))

	// Commands
	serverCmds := commands.Generate(teamserver, teamclient)

	// Run
	carapace.Gen(serverCmds)

	err = serverCmds.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

// mainNoCommands illustrates the fact (without much proof and code) that the teamclient
// and teamserver toolsets are not restrained to CLI usage nor have any obligation to use
// and expose themselves via a CLI.
// On the other hand, some programs may wish to offer it in specific circumstances, or even
// make use of it on the teamclient-side but not on the teamserver. Many setups are possible.
func mainNoCommands() {
	// Server
	gTeamserver := grpc.NewListener()

	teamserver, err := server.New("teamserver", server.WithListener(gTeamserver))
	if err != nil {
		log.Fatal(err)
	}

	// Note that we don't create a self-client for the teamserver: we don't need to have
	// any teamclient interaction with the teamserver, and we just want to start/stop it
	// from our code.
	//
	// Instead, let's first start a listener job on some address: this call is non blocking,
	// and should we want to keep control of the listener job, we can use the returned ID.
	listenerID, err := teamserver.ServeAddr("grpc/mTLS", "localhost", 31350)
	if err != nil {
		log.Fatal(err)
	}

	// We can kill the listener from code like this.
	err = teamserver.ListenerClose(listenerID)
	if err != nil {
		log.Fatal(err)
	}

	// Finally, simply ask the server to start the daemon (blocking), which also starts
	// all listeners that might be saved as persistent jobs. To be noted, you will typically
	// favor the above ServeAddr() function in your code rather than the daemon one below,
	// since -while being entirely possible- the latter will likely be favored by CLI users.
	//
	// Note that we don't pass the name of the listener stack we want to use: the daemon
	// function always uses the first listener backend that has been registered to the server.
	err = teamserver.ServeDaemon("localhost", 31350)
	if err != nil {
		log.Fatal(err)
	}
}

// mainIntegrated demonstrates a use case where the library user might already have an existing,
// established and/or working program. This program will naturally already dispose of core things
// like loggers, database configurations or backends, specific directories for output, etc.
//
// This example therefore shows how to use some other options to tightly integrate the teamserver
// toolset to such programs, while maintaining a strictly identical behavior and function set.
//
// Note that we use nil pointers everywhere in those functions, so this function is very much
// unsafe to run as is. It should be noted again, however, that the library tries to fail safe
// and as early as possible, as illustrated by the various errors returned in examples above.
func mainIntegrated() {
	// Use the classic gRPC example backend.
	gTeamserver := grpc.NewListener()

	var serverOpts []server.Options
	serverOpts = append(serverOpts,
		// Filesystem
		server.WithHomeDirectory("~/.config"), // If we use an appdirectory different from ~/.app/directory .
		server.WithTeamDirectory(""),          // We might want the teamserver-specific output not to use a specific subdir in it.

		// Logging.
		server.WithLogger(nil),                 // We might have a fully set-up logger, with multiple output destinations.
		server.WithLogFile("path/to/log.file"), // Or we are fine with default teamserver logger, but a specific file.

		// Network (listeners and settings).
		server.WithDefaultPort(31340),    // Default port of daemon/listeners.
		server.WithListener(gTeamserver), // Please see above examples, and the documentation. Any number of them can be registered.
		server.WithListener(nil),         // Another listener/RPC backend stack used/needed by your application.

		// Database (stores users certificates)
		server.WithDatabase(nil),       // Either pass the teamserver a running DB to store/fetch users certificates data.
		server.WithDatabaseConfig(nil), // Or a specific configuration to use for connecting to one.
	)

	// Pass those options at creation time: some of them cannot be passed later,
	// while others can (eg, listener backends can be added and listener configs
	// chosen at any time).
	teamserver, err := server.New("teamserver", serverOpts...)
	if err != nil {
		log.Fatal(err)
	}

	// Again, note that we don't pass the name of the listener stack we want to use: the daemon
	// function always uses the first listener backend that has been registered to the server.
	err = teamserver.ServeDaemon("localhost", 31350)
	if err != nil {
		log.Fatal(err)
	}
}

// mainInMemory adapts the mainSmallest example with options to instruct the teamserver
// to never touch the host filesystem: all filesystem calls are redirected to an in-memory
// filesystem (which therefore holds all log files and contents in memory), and an in-memory
// SQLite database instance.
func mainInMemory() {
	var serverOpts []server.Options
	serverOpts = append(serverOpts,
		server.WithInMemory(),
		server.WithDefaultPort(31340), // Default port of daemon/listeners.
	)

	// Pass those options at creation time: some of them cannot be passed later,
	// while others can (eg, listener backends can be added and listener configs
	// chosen at any time).
	teamserver, err := server.New("teamserver", serverOpts...)
	if err != nil {
		log.Fatal(err)
	}

	// Pass specific options for the teamserver
	// self-client, to provide identical behavior.
	var clientOpts []client.Options
	clientOpts = append(clientOpts,
		client.WithInMemory(),
	)

	// Ask the teamserver to create its own teamclient (without any RPC client backend).
	teamclient := teamserver.Self(clientOpts...)

	// Generate a tree of server-side commands: this tree also has client-only
	// commands as a subcommand "client" of the "teamserver" command root here.
	serverCmds := commands.Generate(teamserver, teamclient)

	// Generate completions for the tree.
	carapace.Gen(serverCmds)

	// Run our teamserver binary.
	err = serverCmds.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
