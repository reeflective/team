package transport

import (
	"context"
)

type TokenAuth string

// Return value is mapped to request headers.
func (t TokenAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": "Bearer " + string(t),
	}, nil
}

func (TokenAuth) RequireTransportSecurity() bool {
	return true
}
