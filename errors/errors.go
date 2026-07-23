package errors

import "errors"

var ErrMethodNotAllowed = errors.New("method not allowed")

type JSONError struct {
	Error string `json:"error,omitempty"`
}

// Semantic error categories returned by the domain handlers. They carry no HTTP
// knowledge — `statusFor“ is the single place that maps them to status
// codes — so the same errors could drive a non-HTTP transport (e.g. a consumer)
// unchanged.
var (
	// ErrInvalidInput signals malformed or invalid caller input (maps to 400).
	ErrInvalidInput = errors.New("invalid input")
	// ErrNotFound signals a requested resource does not exist or is
	// unavailable (maps to 404).
	ErrNotFound = errors.New("not found")
)
