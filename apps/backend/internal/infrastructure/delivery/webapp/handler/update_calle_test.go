package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type updateCalleStub struct {
	calle      domain.Calle
	err        error
	called     bool
	gotLoteoID string
	gotCalleID string
	gotData    domain.CalleData
}

func (stub *updateCalleStub) Execute(
	_ context.Context,
	_ loteos.Actor,
	loteoID, calleID string,
	data domain.CalleData,
) (domain.Calle, error) {
	stub.called = true
	stub.gotLoteoID = loteoID
	stub.gotCalleID = calleID
	stub.gotData = data
	return stub.calle, stub.err
}

func performUpdateCalleRequest(t *testing.T, updateCalle *updateCalleStub, verifier userVerifierStub, token, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewUpdateCalleHandler(updateCalle)
	requireAuth := middleware.RequireAuth(verifier)
	mux := http.NewServeMux()
	mux.Handle("PATCH /api/v1/loteos/{loteoId}/calles/{calleId}", requireAuth(handler.Adapt(h, 5*time.Second)))
	return performRequest(t, mux, http.MethodPatch, path, token, body)
}

const calleRoute = "/api/v1/loteos/loteo-1/calles/ca-1"

func calleDataBody() map[string]any {
	return map[string]any{
		"nombre": "Los Álamos",
		"tipo":   "asfalto",
	}
}

func TestUpdateCalleRoute(t *testing.T) {
	t.Parallel()

	t.Run("updates the calle and answers with it", func(t *testing.T) {
		t.Parallel()

		updateCalle := &updateCalleStub{calle: domain.Calle{
			ID: "ca-1", Name: "Los Álamos", Type: domain.CalleTypeAsfalto,
		}}
		recorder := performUpdateCalleRequest(t, updateCalle, administradorVerifier(), "valid-token", calleRoute, calleDataBody())

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if updateCalle.gotLoteoID != "loteo-1" || updateCalle.gotCalleID != "ca-1" {
			t.Errorf("path ids = %q %q", updateCalle.gotLoteoID, updateCalle.gotCalleID)
		}
		if updateCalle.gotData.Name != "Los Álamos" || updateCalle.gotData.Type != "asfalto" {
			t.Errorf("data = %#v", updateCalle.gotData)
		}

		var got domain.Calle
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != "ca-1" || got.Name != "Los Álamos" || got.Type != "asfalto" {
			t.Errorf("response = %#v", got)
		}
	})

	t.Run("maps use case errors to the right status", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "not authorized", err: domain.ErrNoAutorizado, wantStatus: http.StatusForbidden, wantCode: "forbidden"},
			{name: "not found", err: domain.ErrCalleNotFound, wantStatus: http.StatusNotFound, wantCode: "calle_not_found"},
			{name: "missing name", err: domain.ErrInvalidCalleName, wantStatus: http.StatusBadRequest, wantCode: "invalid_calle_nombre"},
			{name: "invalid type", err: domain.ErrInvalidCalleType, wantStatus: http.StatusBadRequest, wantCode: "invalid_calle_tipo"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()
				updateCalle := &updateCalleStub{err: test.err}
				recorder := performUpdateCalleRequest(t, updateCalle, administradorVerifier(), "valid-token", calleRoute, calleDataBody())
				if recorder.Code != test.wantStatus {
					t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
				}
				var got response.ErrorResponse
				if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if got.Code != test.wantCode {
					t.Errorf("error code = %q, want %q", got.Code, test.wantCode)
				}
			})
		}
	})

	t.Run("rejects unauthenticated callers", func(t *testing.T) {
		t.Parallel()
		updateCalle := &updateCalleStub{}
		recorder := performUpdateCalleRequest(t, updateCalle, userVerifierStub{}, "", calleRoute, calleDataBody())
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if updateCalle.called {
			t.Error("use case should not be called without a token")
		}
	})
}
