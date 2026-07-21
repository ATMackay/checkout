package services

import (
	"context"

	"github.com/julienschmidt/httprouter"
)

type Service interface {
	Start(context.Context) error
	Stop() error
	RegisterHandlers() *httprouter.Router
}
