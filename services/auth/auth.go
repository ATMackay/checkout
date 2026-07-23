// Package auth resolves a credential to a caller identity. It is transport-
// neutral: it knows nothing about HTTP headers or status codes. The HTTP
// middleware extracts a credential from the request and calls Authenticate; when
// this graduates to JWT, only the Authenticator implementation changes, not this
// contract nor its callers.
package auth

import (
	"context"
	"errors"
)

// XAuthHeaderKey is the request header carrying the shared password in the
// pre-JWT auth mode. The header is an HTTP detail owned here, not by the
// transport-neutral auth package.
const XAuthHeaderKey = "X-Auth-Password"

// ErrUnauthenticated is returned when a credential does not resolve to a user.
var ErrUnauthenticated = errors.New("unauthenticated")

// Authenticator resolves an opaque credential (a shared password today, a JWT
// later) to a user identifier.
//
//go:generate mockgen -destination mock/auth.go -package mock github.com/ATMackay/checkout/services/auth Authenticator
type Authenticator interface {
	Authenticate(credential string) (userID string, err error)
}

// PasswordAuthenticator is a static credential store mapping shared passwords to
// user IDs. It is the simple password mode that precedes real token auth.
type PasswordAuthenticator struct {
	users map[string]string // password -> userID
}

// NewPasswordAuthenticator builds a PasswordAuthenticator from a
// password -> userID map.
func NewPasswordAuthenticator(users map[string]string) *PasswordAuthenticator {
	return &PasswordAuthenticator{users: users}
}

// Authenticate returns the user ID mapped to credential, or ErrUnauthenticated.
func (a *PasswordAuthenticator) Authenticate(credential string) (string, error) {
	userID, ok := a.users[credential]
	if !ok {
		return "", ErrUnauthenticated
	}
	return userID, nil
}

// contextKey is unexported so no other package can collide with our context
// value under the same key.
type contextKey struct{}

var userIDKey contextKey

// WithUserID returns a copy of ctx carrying the authenticated user ID.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserID returns the authenticated user ID stored in ctx, if present.
func UserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}
