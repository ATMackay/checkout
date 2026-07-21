//go:build !integration

package orders

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ATMackay/checkout/database/mock"
	ordersmock "github.com/ATMackay/checkout/services/orders/mock"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_ServiceMethods(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := mock.NewMockDatabase(ctrl)

	// The relay stays healthy across every case so only the mocked db drives the
	// health outcome; Health pings it on each probe.
	relay := ordersmock.NewMockRelayer(ctrl)
	relay.EXPECT().Ping(gomock.Any()).Return(nil).AnyTimes()

	s := NewService(db, "", relay)

	ctx := context.Background()
	tests := []struct {
		name                 string
		preparefunc          func(*mock.MockDatabase)
		method               string
		path                 string
		body                 []byte
		paramFunc            func() httprouter.Params
		handlerFunc          httprouter.Handle
		expectedResponseCode int
	}{
		{
			"status",
			func(*mock.MockDatabase) {},
			http.MethodGet,
			StatusEndPnt,
			nil,
			func() httprouter.Params { return nil },
			Status(),
			http.StatusOK,
		},
		{
			"health",
			func(md *mock.MockDatabase) {
				md.EXPECT().Ping(ctx).Return(nil)
			},
			http.MethodGet,
			HealthEndPnt,
			nil,
			func() httprouter.Params { return nil },
			s.Health(),
			http.StatusOK,
		},
		{
			"health-ping-error",
			func(md *mock.MockDatabase) {
				md.EXPECT().Ping(ctx).Return(assert.AnError)
			},
			http.MethodGet,
			HealthEndPnt,
			nil,
			func() httprouter.Params { return nil },
			s.Health(),
			http.StatusServiceUnavailable,
		},
		// Add more tests here
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			tc.preparefunc(db)

			req, err := http.NewRequest(tc.method, tc.path, bytes.NewReader(tc.body))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()

			tc.handlerFunc(rr, req, tc.paramFunc())

			assert.Equal(t, tc.expectedResponseCode, rr.Code)
		})
	}
}
