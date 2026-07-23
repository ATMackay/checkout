package services

import (
	"context"

	"github.com/julienschmidt/httprouter"
)

// Service is the transport-agnostic contract every node implements. It carries
// no auth (or any other cross-cutting) dependency: a service applies its own
// per-route concerns when it builds its router.
type Service interface {
	Start(context.Context) error
	Stop() error
	RegisterHandlers() *httprouter.Router
}
