package tenant

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrMissing = errors.New("tenant identity missing from context")

type Identity struct {
	TenantID     uuid.UUID
	UserID       string
	ServiceID    string
	Capabilities map[string]bool
}

func (i Identity) Has(capability string) bool {
	return i.Capabilities[capability]
}

type contextKey struct{}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, identity)
}

func FromContext(ctx context.Context) (Identity, error) {
	identity, ok := ctx.Value(contextKey{}).(Identity)
	if !ok || identity.TenantID == uuid.Nil {
		return Identity{}, ErrMissing
	}
	return identity, nil
}
