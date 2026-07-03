package errors

import "errors"

var ErrMethodNotAllowed = errors.New("method not allowed")

type JSONError struct {
	Error string `json:"error,omitempty"`
}
