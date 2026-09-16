package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
)

func TestLive(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	handler.Live(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}
