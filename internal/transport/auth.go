package transport

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
)

const (
	// TokenLength is used for user tokens, as well as listeners IDs.
	TokenLength = 32
)

// TokenAuth extracts authentication metadata from contexts,
// specifically the "Authorization": "Bearer" key:value pair.
type TokenAuth string

// Return value is mapped to request headers.
func (t TokenAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": "Bearer " + string(t),
	}, nil
}

// RequireTransportSecurity always return true.
func (TokenAuth) RequireTransportSecurity() bool {
	return true
}
