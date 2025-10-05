package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

// Authentication middleware - password in Header.

// authMiddleware adds password protection to specific hhtp routes
func (s *Server) authMiddleware(h httprouter.Handle) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		// Get the password from the header
		password := req.Header.Get("X-Auth-Password")

		if password != s.authPassword {
			// Return 401 Unauthorized if the password is incorrect
			respondWithError(w, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
			return
		}

		h(w, req, p)
	})
}

// Observability middleware

// observerMiddleware provides logging and metrics middleware, surfacing low level request/response data from the http server.
func observerMiddleware(h httprouter.Handle) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {

		statusRecorder := &responseRecorder{ResponseWriter: w}

		start := time.Now()
		h(statusRecorder, req, p)
		elapsed := time.Since(start)

		httpCode := statusRecorder.statusCode
		// prometheus metrics
		RequestDuration.WithLabelValues(req.Method, req.URL.Path, http.StatusText(httpCode)).Observe(elapsed.Seconds())
		RequestCount.WithLabelValues(req.Method, req.URL.Path, http.StatusText(httpCode)).Inc()

		// only log full request/response data if running in debug mode or if
		// the server returned an error response code.
		if httpCode > 399 {
			slog.Warn("http Err",
				"http_method", req.Method,
				"http_code", httpCode,
				"elapsed", elapsed.Milliseconds(),
				"url", req.URL.Path,
			)
		} else {
			slog.Debug("served Http Request",
				"http_method", req.Method,
				"http_code", httpCode,
				"elapsed", elapsed.Milliseconds(),
				"url", req.URL.Path)
		}
	})
}

// responseRecorder is a wrapper for http.ResponseWriter used
// by logging middleware.
type responseRecorder struct {
	http.ResponseWriter

	statusCode int
	response   []byte
}

func (w *responseRecorder) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseRecorder) Write(b []byte) (int, error) {
	w.response = b
	return w.ResponseWriter.Write(b)
}
