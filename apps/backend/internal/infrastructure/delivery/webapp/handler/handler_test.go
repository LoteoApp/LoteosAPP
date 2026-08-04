package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/route"
)

type serviceStub struct {
	info     domain.Info
	infoErr  error
	readyErr error
}

func (service serviceStub) Info(context.Context) (domain.Info, error) {
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

			recorder := performRequest(test.service, test.path)
			if recorder.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}

func TestInfoRoute(t *testing.T) {
	t.Parallel()

	want := domain.Info{Service: "loteosapp-backend", Status: "ok"}
	recorder := performRequest(serviceStub{info: want}, "/api/v1/system")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var got domain.Info
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Service != want.Service || got.Status != want.Status {
		t.Errorf("response = %#v, want %#v", got, want)
	}
}

func TestInfoRouteHidesInternalErrors(t *testing.T) {
	t.Parallel()

	recorder := performRequest(
		serviceStub{infoErr: errors.New("connection password leaked")},
		"/api/v1/system",
	)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}

	var got response.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
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
	h := handler.NewHandler(service)
	mux := http.NewServeMux()
	route.RegisterRoutes(mux, h)

	request := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	return recorder
}
