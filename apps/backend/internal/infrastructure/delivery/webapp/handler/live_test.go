package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"loteosapp/backend/internal/infrastructure/delivery/webapp/route"
)

func TestLiveRoute(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	route.RegisterRoutes(mux, nil, nil, nil, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}
