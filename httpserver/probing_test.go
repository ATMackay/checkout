package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ATMackay/checkout/model"
)

func TestStatusHandler(t *testing.T) {
	rr := httptest.NewRecorder()
	StatusHandler("svc", "1.2.3")(rr, httptest.NewRequest(http.MethodGet, "/status", nil), nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rr.Code)
	}
	var resp model.StatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Service != "svc" || resp.Version != "1.2.3" {
		t.Errorf("resp = %+v, want service=svc version=1.2.3", resp)
	}
}

func TestHealthHandler(t *testing.T) {
	ok := func(context.Context) error { return nil }
	fail := func(context.Context) error { return errors.New("down") }

	tests := []struct {
		name         string
		checks       []Check
		wantCode     int
		wantFailures int
	}{
		{"all pass", []Check{{"db", ok}, {"broker", ok}}, http.StatusOK, 0},
		{"one fails", []Check{{"db", ok}, {"broker", fail}}, http.StatusServiceUnavailable, 1},
		{"all fail", []Check{{"db", fail}, {"broker", fail}}, http.StatusServiceUnavailable, 2},
		{"no checks", nil, http.StatusOK, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			HealthHandler("svc", "1.0", tt.checks...)(rr, httptest.NewRequest(http.MethodGet, "/health", nil), nil)

			if rr.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d", rr.Code, tt.wantCode)
			}
			var resp model.HealthResponse
			if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(resp.Failures) != tt.wantFailures {
				t.Errorf("failures = %v, want %d", resp.Failures, tt.wantFailures)
			}
		})
	}
}
