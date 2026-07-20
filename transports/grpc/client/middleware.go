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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/reeflective/team/client"
)

// ErrNoTLSCredentials is returned when TLS dial options are requested but the
// selected teamserver config has no credentials (e.g. an in-memory dialer).
var ErrNoTLSCredentials = errors.New("the teamclient has no TLS credentials to use")

// TLSAuthMiddleware returns the Mutual-TLS dial options for the teamclient's
// selected remote server configuration: transport credentials built from the
// config's CA/cert/key, plus a per-RPC bearer token carrying the config token.
func TLSAuthMiddleware(cli *client.Client) ([]grpc.DialOption, error) {
	config := cli.Config()
	if config == nil || config.PrivateKey == "" {
		return nil, ErrNoTLSCredentials
	}

	tlsConfig, err := cli.NewTLSConfigFrom(config.CACertificate, config.Certificate, config.PrivateKey)
	if err != nil {
		return nil, err
	}

	return []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithPerRPCCredentials(tokenAuth(config.Token)),
	}, nil
}

// tokenAuth is a credentials.PerRPCCredentials that sends the teamclient's API
// token as an "Authorization: Bearer <token>" header on every request. The
// server's token-authentication interceptor reads it (see the server transport).
type tokenAuth string

// GetRequestMetadata maps the token to the Authorization request header.
func (t tokenAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": "Bearer " + string(t),
	}, nil
}

// RequireTransportSecurity always returns true: the bearer token is only ever
// sent over the Mutual-TLS transport established above.
func (tokenAuth) RequireTransportSecurity() bool {
	return true
}
