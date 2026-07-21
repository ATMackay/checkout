// Package middleware holds HTTP middleware shared by every service that runs on
// the common httpserver. Nothing here is domain-specific: Observer is applied by
// the server to all traffic, and Auth is a reusable per-route guard.
package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ATMackay/checkout/errors"
	"github.com/julienschmidt/httprouter"
)

// Observer wraps an http.Handler with request logging and Prometheus metrics.
// The httpserver applies it once around the whole router, so every service —
// orders today, the notifier next — gets identical observability for free rather
// than opting in per route.
func Observer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Default to 200: a handler that writes a body without an explicit
		// WriteHeader still reports the status the client actually receives.
		rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		start := time.Now()
		next.ServeHTTP(rec, req)
		elapsed := time.Since(start)

		status := http.StatusText(rec.statusCode)
		RequestDuration.WithLabelValues(req.Method, req.URL.Path, status).Observe(elapsed.Seconds())
		RequestCount.WithLabelValues(req.Method, req.URL.Path, status).Inc()

		attrs := []any{
			"http_method", req.Method,
			"http_code", rec.statusCode,
			"elapsed_us", elapsed.Microseconds(),
			"url", req.URL.Path,
		}
		// Only warn on error responses; everything else is debug-level detail.
		if rec.statusCode > 399 {
			slog.Warn("http err", attrs...)
		} else {
			slog.Debug("served http request", attrs...)
		}
	})
}

// Auth returns middleware that rejects requests whose X-Auth-Password header does
// not match password. It is deliberately per-route: services wrap the handlers
// that need protection and leave probes and metrics open.
func Auth(password string) func(httprouter.Handle) httprouter.Handle {
	return func(h httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
			if req.Header.Get("X-Auth-Password") != password {
				writeJSONError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			h(w, req, p)
		}
	}
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errors.JSONError{Error: msg})
}

// responseRecorder captures the status code written downstream so Observer can
// record it.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseRecorder) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
