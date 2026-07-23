package httpserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ATMackay/checkout/model"
	"github.com/julienschmidt/httprouter"
)

var (
	StatusEndPnt = "/status"
	HealthEndPnt = "/health"
)

// Check is a named readiness probe for one dependency. Services declare a Check
// per backend they depend on; HealthHandler runs them all.
type Check struct {
	Name  string
	Probe func(context.Context) error
}

// StatusHandler serves a static liveness response — always 200 while the process
// is up. Shared by every service so the /status contract is identical.
func StatusHandler(service, version string) httprouter.Handle {
	return Handle(func(*http.Request, httprouter.Params) (any, error) {
		return &model.StatusResponse{Message: "OK", Version: version, Service: service}, nil
	})
}

// HealthHandler serves a readiness report: it runs every check and returns 503
// with the failure list when any fail, else 200. This is the single home of the
// report-body-with-non-200 shape (which the Handle adapter can't express), so
// each service supplies only its own list of checks rather than reimplementing
// the mechanism.
func HealthHandler(service, version string, checks ...Check) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		report := &model.HealthResponse{Service: service, Version: version}
		failures := []string{}
		for _, c := range checks {
			if err := c.Probe(r.Context()); err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", c.Name, err))
			}
		}
		report.Failures = failures

		code := http.StatusOK
		if len(failures) > 0 {
			code = http.StatusServiceUnavailable
		}
		if err := WriteJSON(w, code, report); err != nil {
			slog.Error("failed to write health response", "error", err)
		}
	}
}
