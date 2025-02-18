package server

import (
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// Authentication Middleware  - TODO
func (s *HTTPServer) authMiddleware(h httprouter.Handle) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {
		// Get the password from the header
		password := req.Header.Get("X-Auth-Password")

		// Check if the password is correct
		if password != s.authPassword {
			// Return 401 Unauthorized if the password is incorrect
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized: Invalid password"))
			return
		}

		// Call the next handler if authentication succeeds
		h(w, req, p)
	})
}

// HTTP logging middleware

// logHTTPMiddleware provides logging middleware, surfacing low level request/response data from the http server.
func logHTTPMiddleware(h httprouter.Handle) httprouter.Handle {
	return httprouter.Handle(func(w http.ResponseWriter, req *http.Request, p httprouter.Params) {

		statusRecorder := &responseRecorder{ResponseWriter: w}

		start := time.Now()
		h(statusRecorder, req, p)
		elapsed := time.Since(start)

		httpCode := statusRecorder.statusCode
		entry := logrus.WithFields(logrus.Fields{
			"http_method":          req.Method,
			"http_code":            httpCode,
			"elapsed_microseconds": elapsed.Microseconds(),
			"url":                  req.URL.Path,
		})
		// only log full request/response data if running in debug mode or if
		// the server returned an error response code.
		if httpCode > 399 {
			entry.Warn("httpErr")
		} else {
			entry.Debug("servedHttpRequest")
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
