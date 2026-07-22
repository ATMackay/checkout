//go:build !integration

package services

import (
	"net/http"
	"testing"

	"github.com/julienschmidt/httprouter"
)

func TestRegisterHandlers(t *testing.T) {
	a := AddEndpoints([]EndPoint{{Path: "/foo", Handler: func(http.ResponseWriter, *http.Request, httprouter.Params) {}, MethodType: http.MethodGet}})
	r := a.Routes()
	if r == nil {
		t.Fatal("should not return nil router")
	}
}
