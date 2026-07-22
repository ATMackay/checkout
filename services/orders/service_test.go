//go:build !integration

package orders

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ATMackay/checkout/database/mock"
	"github.com/ATMackay/checkout/services/auth"
	ordersmock "github.com/ATMackay/checkout/services/orders/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// Test_ServiceProbes exercises the orders service's status/health wiring through
// its real router: /status is always 200, /health is 200 when its checks pass
// and 503 when the database probe fails. The health mechanism itself is covered
// in httpserver; this verifies orders supplies the right checks.
func Test_ServiceProbes(t *testing.T) {
	ctrl := gomock.NewController(t)

	db := mock.NewMockDatabase(ctrl)
	relay := ordersmock.NewMockRelayer(ctrl)
	relay.EXPECT().Ping(gomock.Any()).Return(nil).AnyTimes() // broker stays healthy

	s := NewService(db, relay, auth.NewPasswordAuthenticator(nil))
	router := s.RegisterHandlers()

	tests := []struct {
		name     string
		prepare  func(*mock.MockDatabase)
		path     string
		wantCode int
	}{
		{"status", func(*mock.MockDatabase) {}, StatusEndPnt, http.StatusOK},
		{"health", func(md *mock.MockDatabase) {
			md.EXPECT().Ping(gomock.Any()).Return(nil)
		}, HealthEndPnt, http.StatusOK},
		{"health-db-down", func(md *mock.MockDatabase) {
			md.EXPECT().Ping(gomock.Any()).Return(assert.AnError)
		}, HealthEndPnt, http.StatusServiceUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.prepare(db)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
			assert.Equal(t, tc.wantCode, rr.Code)
		})
	}
}
