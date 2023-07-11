package transport

import (
	"context"
)

const (
	// TokenLength is used for user tokens, as well as listeners IDs.
	TokenLength = 32
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
