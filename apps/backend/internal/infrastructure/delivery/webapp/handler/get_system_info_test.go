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

type getSystemInfoStub struct {
	info domain.Info
	err  error
}

func (stub getSystemInfoStub) Execute(context.Context) (domain.Info, error) {
	return stub.info, stub.err
}

func performSystemInfoRequest(stub getSystemInfoStub) *httptest.ResponseRecorder {
	h := handler.NewGetSystemInfoHandler(stub)
	mux := http.NewServeMux()
	route.RegisterRoutes(mux, h, nil, nil, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/system", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	return recorder
}

func TestInfoRoute(t *testing.T) {
	t.Parallel()

	want := domain.Info{Service: "loteosapp-backend", Status: "ok"}
	recorder := performSystemInfoRequest(getSystemInfoStub{info: want})

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
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

	recorder := performSystemInfoRequest(getSystemInfoStub{err: &domain.Error{
		Kind:    domain.KindUnavailable,
		Code:    "database_diagnostic_failed",
		Message: "No se pudo consultar PostgreSQL",
		Cause:   errors.New("connection password leaked"),
	}})

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
