
<div align="center">
  <br> <h1> Team </h1>

  <p>  Transform any Go program into a client of itself, remotely or locally.  </p>
  <p>  Use, manage teamservers and clients with code, with their CLI, or both.  </p>
</div>


<!-- Badges -->
<!-- Assuming the majority of them being written in Go, most of the badges below -->
<!-- Replace the repo name: :%s/reeflective\/template/reeflective\/repo/g -->

<p align="center">
  <a href="https://github.com/reeflective/team/actions/workflows/go.yml">
    <img src="https://github.com/reeflective/team/actions/workflows/go.yml/badge.svg?branch=main"
      alt="Github Actions (workflows)" />
  </a>

  <a href="https://github.com/reeflective/team">
    <img src="https://img.shields.io/github/go-mod/go-version/reeflective/team.svg"
      alt="Go module version" />
  </a>

  <a href="https://pkg.go.dev/github.com/reeflective/team">
    <img src="https://img.shields.io/badge/godoc-reference-blue.svg"
      alt="GoDoc reference" />
  </a>

  <a href="https://goreportcard.com/report/github.com/reeflective/team">
    <img src="https://goreportcard.com/badge/github.com/reeflective/team"
      alt="Go Report Card" />
  </a>

  <a href="https://codecov.io/gh/reeflective/team">
    <img src="https://codecov.io/gh/reeflective/team/branch/main/graph/badge.svg"
      alt="codecov" />
  </a>

  <a href="https://opensource.org/licenses/BSD-3-Clause">
    <img src="https://img.shields.io/badge/License-BSD_3--Clause-blue.svg"
      alt="License: BSD-3" />
  </a>
</p>


-----
## Summary

The client-server paradigm is an ubiquitous concept in computer science. Equally large and common is
the problem of building software that _collaborates_ easily with other peer programs. Although
writing collaborative software seems to be the daily task of many engineers around the world,
succeedingly and easily doing so in big programs as well as in smaller ones is not more easily done
than said. Difficulty still increases -and keeping in mind that humans use software and not the
inverse- when programs must enhance the capacity of humans to collaborate while not restricting the
number of ways they can do so, for small tasks as well as for complex ones.

The `reeflective/team` library provides a small toolset for arbitrary programs (and especially those
controlled in more or less interactive ways) to collaborate together by acting as clients and
servers of each others, as part of a team. Teams being made of players (humans _and_ their tools),
the library focuses on offering a toolset for "human teaming": that is, treating software tools that
are either _teamclients_ or _teamservers_ of others, within a defined -generally refrained- team of
users, which shall generally be strictly and securely authenticated.

The project originates from the refactoring of a security-oriented tool that used this approach to
clearly segregate client and server binary code (the former's not needing most of the latter's).
Besides, the large exposure of the said-tool to the CLI prompted the author of the
`reeflective/team` library to rethink how the notion of "collaborative programs" could be approached
and explored from different viewpoints: distinguishing between the tools' developers, and their
users. After having to reuse this core code for other projects, the idea appeared to extract the
relevant parts and to restructure and repackage them behind coherent interfaces (API and CLI).

The result of this refactoring consists in 2 Go packages (`client` and `server`) for programs needing to
act as:
- A **Team client**: a program, or one of its components, that needs to rely on a "remote" program peer
  to serve some functionality that is available to a team of users' tools. The program acting as a
  _teamclient_ may do so for things as simple as sending a message to the team, or as complicated as a
  compiler backend with which multiple client programs can send data to process and build.
- A **Team server**: The remote, server-side counterpart of the software teamclient. Again, the
  teamserver can be doing anything, from simply notifying users' teamclient connections to all the team
  all the way to handling very complex and resource-hungry tasks that can only be ran on a server host.

Throughout this library and its documentation, various words are repeatedly employed:
- _teamclient_ refers to either the client-specific toolset provided by this library
  (`team/client.Client` core type) or the software making use of this teamclient code.
- _teamserver_ refers to either the server-specific toolset provided to make a program serve its
  functionality remotely, or to the tools embedding this code in order to do so.
- _team tool/s_ might be used to refer to programs using either or all of the library components at
  large.

-----
## Purposes, Constraints and Features

The library rests on several principles, constraints and ideas to fulfill its intended purpose:
- The library's sole aim is to **make most programs able to collaborate together** under the
  paradigm of team clients and team servers, and to do so while ensuring performance, coherence,
  ease of use and security of all processes and workflows involved. This, under the _separate
  viewpoints_ of tool development, enhancement and usage.
- Ensure a **working-by-default toolset**, assuming that the time spent on any tool's configuration
  is inversely proportional to its usage. Emphasis on this aspect should apply equally well to team
  tools' users and developers.
- Ensure the **full, secure and reliable authentication of all team clients and servers'
  interactions**, by using certificate-based communication encryption and user authentication, _aka_
  a "zero-trust" model. Related and equally important, ensure the various team toolset interfaces
  provide for easy and secure usage of their host tools.
- **Accomodate for the needs of developers to use more specific components**, at times or at points,
  while not hampering on the working-by-default aspects of the team client/server toolset. Examples
  include replacing parts or all of the transport, RPC, loggers, database and filesystem
  backends.
- To that effect, the library **offer different interfaces to its functionality**: an API (Go code)
  aiming to provide developers a working-by-default, simple but powerful way to instruct their
  software how to collaborate with peers.

Related or resulting from the above, below are examples of behavior adopted by this library:
- All errors returned by the API are always logged before return (with configured log behavior).
- Interactions with the filesystem restrained until they need to happen.
- The default database is a pure Go file-based sqlite db, which can be configured to run in memory.
- Unless absolutely needed or specified otherwise, return all critical errors instead of log
  fatal/panicking (exception made of the certificate infrastructure which absolutely needs to work
  for security reasons).

-----
## CLI examples (users)

-----
## API examples (developers)

-----
## Documentation

-----
## Differences with Hashicorp Go plugin system

At first glance, different and not much related to our current topic is the equally large problem of
dynamic code loading and execution for arbitrary programs. In the spectrum of major programming
languages, various approaches have been taken to tackle the dynamic linking, loading and execution
problem, with interpreted languages offering the most common solutioning approach to this.

The Go language (and many other compiled languages that do not encourage dynamic linking for that
matter) has to deal with the problem through other means, the first of which simply being the
adoption of different architectural designs in the first place (eg. "microservices"). Another path
has been the "plugin system" for emulating the dynamic workflows of interpreted languages, of which
the most widely used attempt being the [Hashicorp plugin
system](https://github.com/hashicorp/go-plugin), which entirely rests on an (g)RPC backend.


-----
## Status

The Command-Line and Application-Programming Interfaces of this library are unlikely to change
much in the future, and should be considered mostly stable. These might grow a little bit, but
will not shrink, as they been already designed to be as minimal as they could be.

In particular, `client.Options` and `server.Options` APIs might grow, so that new features/behaviors
can be integrated without the need for the teamclients and teamservers types APIs to change.

The section **Possible Enhancements** below includes 9 points, which should grossly be equal
to 9 minor releases (`0.1.0`, `0.2.0`, `0.3.0`, etc...), ending up in `v1.0.0`.

- Please open a PR or an issue if you face any bug, it will be promptly resolved.
- New features and/or PRs are welcome if they are likely to be useful to most users.

-----
## Possible enhancements

The list below is not an indication on the roadmap of this repository, but should be viewed as
things the author of this library would be very glad to merge contributions for, or get ideas. 
This teamserver library aims to remain small, with a precise behavior and role.
Overall, contributions and ideas should revolve around strenghening its core/transport code
or around enhancing its interoperability with as much Go code/programs as possible.

- [ ] Use viper for configs.
- [ ] Use afero filesystem.
- [ ] Add support for encrypted sqlite by default.
- [ ] Encrypt in-memory channels, or add option for it.
- [ ] Simpler/different listener/dialer backend interfaces, if it appears needed.
- [ ] Abstract away the client-side authentication, for pluggable auth/credential models.
- [ ] Replace logrus entirely and restructure behind a single package used by both client/server.
- [ ] Review/refine/strenghen the dialer/listener init/close/start process, if it appears needed.
- [ ] `teamclient update` downloads latest version of the server binary + method to `team.Client` for it.

