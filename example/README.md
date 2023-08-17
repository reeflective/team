
# Examples

This directory contains several example components or programs leveraging teamclients and/or teamservers.
Each of the packages' code is documented to the best extent possible, and structured accordingly.
The author hopes that by entering the code by the `main()` function and reading carefully through will
be enough for most library users to get a good first grasp of the `team` library programming model.

Overall, the example directory can serve two purposes:
- To serve as an example usage of the library, with different directories showing how to use
  different components, as well as how they make use the teamserver/clients core in their own way.
- To be outright copy-pasted in your own code structure and modified where needed.
  This should possible to the extent that the code has been rewritten to have sensible
  fallback behavior, to log its various steps and errors, and to fail earlier rather later.

## Exploring examples as a tool user

Since this library is about enabling users to "team-connect" their tools to make them collaborate,
you can also test this library from this user perspective by installing the examples as binaries
and toying with them:

```bash
# Install the teamserver and teamclients.
go install github.com/reeflective/team/example/teamserver
go install github.com/reeflective/team/example/teamclient

# Install the completion engine and source both tools' scripts (optional)
# See this project documentation for setup with your own shell. Below is bash/zsh.
go install github.com/rsteube/carapace-bin@latest
source <(teamserver _carapace)
source <(teamclient _carapace)

# Use the tools
teamserver status
teamserver user --name Michael --host localhost
teamclient import Michael_localhost.teamclient.cfg

teamserver daemon &
teamclient version
```

## I - General workflow & entrypoints

Therefore, users should probably start reading those files in order:
1) `teamserver/main.go` shows several examples of server-side program entrypoints, with varying
   transports, behaviors, backends and options.
2) `teamclient/main.go` shows some equivalent examples for teamclient-only programs. These functions
   are a counterpart to some of those found in (1), as they use the same transport/RPC stack.

## II - Transport backends

### Team Server
Once you feel having a good understanding of entrypoints (server/client creation, setup, CLI generation
and related), you can then take a look at the transport backends mechanism, which allows to register any
number of transport/RPC listener/server backends to your core application teamserver:

3) `transports/grpc/server/server.go` is used by the `teamserver/main.go` to create and prepare a new
   RPC backend. This file is good in that it shows you clearly how to use the teamserver as a core
   driver for any application-specific teamserving process. For example it shows how to query the
   teamserver users, request their connection/token credentials, authenticate them, and log all steps
   with the teamserver loggers.   
4) `transports/grpc/server/middleware.go` shows a quite complex but secure use of gRPC middleware using
   the teamserver authentication and logging toolset. Note that gRPC is a quite beefy stack, and not
   very idiomatic for Go. Don't be too scared at this code if you don't understand it at first, since
   it's very likely that you will either decide to use it as is (or mostly), or that you will opt for
   simpler transport backends to plug onto your teamservers.

### Team Client
5) `transports/grpc/client/client.go` is used by the `teamclient/main.go` to create and prepare the gRPC
   counterpart to the server backend previously mentioned, using similar APIs to setup authentication,
   encryption and logging, and implementing the dialer backend in the same fashion.
6) `transports/grpc/client/middleware.go` is very much identical in form and intents to its
   `transports/grpc/server/middleware.go` counterpart.
