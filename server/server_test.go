package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ATMackay/checkout/database/mock"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_ServerStartStop(t *testing.T) {

	s := NewServer(8001, nil, "")

	s.Start()
	// Wait until server goroutine has initialized
	for !s.started.Load() {
	}

	require.Equal(t, ":8001", s.Addr())

	if err := s.Stop(); err != nil {
		t.Fatal(err)
	}
}

func Test_ServerEndpoints(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	db := mock.NewMockDatabase(ctrl)

	s := NewServer(8001, db, "")

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
		})
	}
}
