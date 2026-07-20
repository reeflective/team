package client

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/reeflective/team"
	"github.com/reeflective/team/client"
	"github.com/reeflective/team/transports/grpc/proto"
)

const (
	kb = 1024
	mb = kb * 1024
	gb = mb * 1024

	// ClientMaxReceiveMessageSize is the max gRPC message size (~2GB).
	ClientMaxReceiveMessageSize = 2*gb - 1

	defaultTimeout = 10 * time.Second
)

// ErrNoConnection is returned when a hook or accessor needs the gRPC client
// connection but Dial() has not (yet) established one.
var ErrNoConnection = errors.New("no gRPC client connection")

// Dialer is a ready-to-use gRPC team/client.Dialer. It is the supported,
// importable evolution of the code that used to live under
// example/transports/grpc/client.
//
// It dials the remote teamserver described by the teamclient's selected config
// (Mutual TLS when the config carries credentials; plaintext for an in-memory
// bufconn dialer whose options were produced by the server transport's
// NewClientFrom). It deliberately registers NO application RPC client of its
// own; instead it EXPOSES the established *grpc.ClientConn so applications can
// build their own service clients on the shared connection:
//
//   - Conn() returns the connection after Dial().
//   - PostDial(hook) runs your hook with the connection right after Dial()
//     succeeds (the counterpart to the server handler's PostServe).
type Dialer struct {
	team    *client.Client
	options []grpc.DialOption
	hooks   []func(*grpc.ClientConn) error
	conn    *grpc.ClientConn
	rpc     proto.TeamClient
}

// NewClient returns a gRPC teamclient dialer loaded with the provided dial
// options (a max-receive-size call option is always added). For in-memory use,
// pass the options returned by the server transport's NewClientFrom.
func NewClient(opts ...grpc.DialOption) *Dialer {
	d := &Dialer{}
	d.options = append(d.options, opts...)
	d.options = append(d.options,
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(ClientMaxReceiveMessageSize)),
	)

	return d
}

// PostDial registers hooks to run, in order, immediately after Dial()
// establishes the connection — the seam for registering application service
// clients on the shared *grpc.ClientConn. A hook error fails the dial.
func (d *Dialer) PostDial(hooks ...func(*grpc.ClientConn) error) {
	d.hooks = append(d.hooks, hooks...)
}

// Conn returns the established gRPC client connection, or nil before Dial().
func (d *Dialer) Conn() *grpc.ClientConn {
	return d.conn
}

// Init implements team/client.Dialer.Init(). It binds the teamclient core and
// assembles dial options from the selected server config: when the config
// carries a private key it adds Mutual-TLS credentials, otherwise it stays
// plaintext (the in-memory case).
func (d *Dialer) Init(cli *client.Client) error {
	d.team = cli
	config := cli.Config()

	// If the configuration has credentials, we are a remote dialer:
	// authenticate and encrypt with Mutual TLS + per-RPC bearer token.
	if config != nil && config.PrivateKey != "" {
		tlsOpts, err := TLSAuthMiddleware(cli)
		if err != nil {
			return err
		}

		d.options = append(d.options, tlsOpts...)
	}

	return nil
}

// Dial implements team/client.Dialer.Dial(). It connects to the configured
// host:port, then runs any PostDial hooks so the application can register its
// service clients on the connection.
func (d *Dialer) Dial() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cfg := d.team.Config()
	host := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	d.conn, err = grpc.DialContext(ctx, host, d.options...)
	if err != nil {
		return err
	}

	// The core Team service client (users/version). It is always wired; calls
	// only reach the wire if the application actually invokes Users()/
	// VersionServer(), and only succeed if the server enabled WithCoreServices().
	d.rpc = proto.NewTeamClient(d.conn)

	for _, hook := range d.hooks {
		if hook == nil {
			continue
		}

		if err := hook(d.conn); err != nil {
			return err
		}
	}

	return nil
}

// Close implements team/client.Dialer.Close(); it closes the connection if any.
func (d *Dialer) Close() error {
	if d.conn == nil {
		return nil
	}

	return d.conn.Close()
}

// Users returns the list of teamserver users, via the core Team service. It
// requires the server to have been created with WithCoreServices(); otherwise
// the call returns an Unimplemented error. Implementing this (and
// VersionServer) makes the dialer satisfy team.Client, so a teamclient created
// WithDialer(this) answers its Users()/VersionServer() through this transport
// automatically.
func (d *Dialer) Users() ([]team.User, error) {
	if d.rpc == nil {
		return nil, ErrNoConnection
	}

	res, err := d.rpc.GetUsers(context.Background(), &proto.Empty{})
	if err != nil {
		return nil, err
	}

	users := make([]team.User, 0, len(res.GetUsers()))
	for _, user := range res.GetUsers() {
		users = append(users, team.User{
			Name:     user.Name,
			Online:   user.Online,
			LastSeen: time.Unix(user.LastSeen, 0),
			Clients:  int(user.Clients),
		})
	}

	return users, nil
}

// VersionServer returns the connected teamserver's version, via the core Team
// service (requires WithCoreServices() on the server).
func (d *Dialer) VersionServer() (team.Version, error) {
	if d.rpc == nil {
		return team.Version{}, ErrNoConnection
	}

	ver, err := d.rpc.GetVersion(context.Background(), &proto.Empty{})
	if err != nil {
		return team.Version{}, errors.New(status.Convert(err).Message())
	}

	return team.Version{
		Major:      ver.Major,
		Minor:      ver.Minor,
		Patch:      ver.Patch,
		Commit:     ver.Commit,
		Dirty:      ver.Dirty,
		CompiledAt: ver.CompiledAt,
		OS:         ver.OS,
		Arch:       ver.Arch,
	}, nil
}

// compile-time guarantees: the dialer is a team client.Dialer, and — because it
// implements Users()/VersionServer() — also a team.Client backend.
var (
	_ client.Dialer = (*Dialer)(nil)
	_ team.Client   = (*Dialer)(nil)
)
