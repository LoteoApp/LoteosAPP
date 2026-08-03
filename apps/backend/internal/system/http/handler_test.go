package systemhttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"loteosapp/backend/internal/platform/httpx"
	"loteosapp/backend/internal/system"
)

type serviceStub struct {
	info     system.Info
	infoErr  error
	readyErr error
}

func (service serviceStub) Info(context.Context) (system.Info, error) {
	return service.info, service.infoErr
}

func (service serviceStub) Ready(context.Context) error {
	return service.readyErr
}

func TestHealthRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		service    serviceStub
		wantStatus int
	}{
		{name: "live", path: "/healthz", wantStatus: http.StatusOK},
		{name: "ready", path: "/readyz", wantStatus: http.StatusOK},
		{
			name:       "database unavailable",
			path:       "/readyz",
			service:    serviceStub{readyErr: errors.New("database unavailable")},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			response := performRequest(test.service, test.path)
			if response.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
}

func TestInfoRoute(t *testing.T) {
	t.Parallel()

	want := system.Info{Service: "loteosapp-backend", Status: "ok"}
	response := performRequest(serviceStub{info: want}, "/api/v1/system")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var got system.Info
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Service != want.Service || got.Status != want.Status {
		t.Errorf("response = %#v, want %#v", got, want)
	}
}

func TestInfoRouteHidesInternalErrors(t *testing.T) {
	t.Parallel()

	response := performRequest(
		serviceStub{infoErr: errors.New("connection password leaked")},
		"/api/v1/system",
	)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}

	var got httpx.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Code != "database_diagnostic_failed" {
		t.Errorf("error code = %q", got.Code)
	}
	if got.Message == "connection password leaked" {
		t.Error("response exposes the internal error")
	}
}

func performRequest(service serviceStub, path string) *httptest.ResponseRecorder {
	handler := NewHandler(service)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	return response
}
