package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/route"
)

type checkDatabaseReadinessStub struct {
	err error
}

func (stub checkDatabaseReadinessStub) Execute(context.Context) error {
	return stub.err
}

func TestReadyRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "database reachable", wantStatus: http.StatusOK},
		{
			name:       "database unavailable",
			err:        errors.New("database unavailable"),
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			h := handler.NewCheckDatabaseReadinessHandler(checkDatabaseReadinessStub{err: test.err})
			mux := http.NewServeMux()
			route.RegisterRoutes(mux, nil, h, nil, nil, nil)

			request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}
