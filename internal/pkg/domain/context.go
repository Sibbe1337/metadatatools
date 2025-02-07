package domain

import (
	"context"
)

type contextKey string

const (
	userContextKey contextKey = "user"
)

// WithUser adds a user to the context
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext retrieves a user from the context
func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userContextKey).(*User)
	return user, ok
}
