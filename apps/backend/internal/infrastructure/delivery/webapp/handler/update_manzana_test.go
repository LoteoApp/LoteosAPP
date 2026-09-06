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

type updateManzanaStub struct {
	manzana      domain.Manzana
	err          error
	called       bool
	gotLoteoID   string
	gotManzanaID string
	gotData      domain.ManzanaData
}

func (stub *updateManzanaStub) Execute(
	_ context.Context,
	_ loteos.Actor,
	loteoID, manzanaID string,
	data domain.ManzanaData,
) (domain.Manzana, error) {
	stub.called = true
	stub.gotLoteoID = loteoID
	stub.gotManzanaID = manzanaID
	stub.gotData = data
	return stub.manzana, stub.err
}

func performUpdateManzanaRequest(t *testing.T, updateManzana *updateManzanaStub, verifier userVerifierStub, token, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewUpdateManzanaHandler(updateManzana)
	requireAuth := middleware.RequireAuth(verifier)
	mux := http.NewServeMux()
	mux.Handle("PATCH /api/v1/loteos/{loteoId}/manzanas/{manzanaId}", requireAuth(handler.Adapt(h, 5*time.Second)))
	return performRequest(t, mux, http.MethodPatch, path, token, body)
}

const manzanaRoute = "/api/v1/loteos/loteo-1/manzanas/mz-1"

func manzanaDataBody() map[string]any {
	return map[string]any{
		"numero":      "A",
		"tieneAgua":   true,
		"tieneCloaca": false,
		"tieneLuz":    true,
		"tieneGas":    false,
		"calleIds":    []string{"calle-1"},
	}
}

func TestUpdateManzanaRoute(t *testing.T) {
	t.Parallel()

	t.Run("updates the manzana and answers with it", func(t *testing.T) {
		t.Parallel()

		updateManzana := &updateManzanaStub{manzana: domain.Manzana{
			ID: "mz-1", Number: "A", HasWater: true, HasPower: true, CalleIDs: []string{"calle-1"},
		}}
		recorder := performUpdateManzanaRequest(t, updateManzana, administradorVerifier(), "valid-token", manzanaRoute, manzanaDataBody())

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if updateManzana.gotLoteoID != "loteo-1" || updateManzana.gotManzanaID != "mz-1" {
			t.Errorf("path ids = %q %q", updateManzana.gotLoteoID, updateManzana.gotManzanaID)
		}
		if !updateManzana.gotData.HasWater || updateManzana.gotData.Number != "A" {
			t.Errorf("data = %#v", updateManzana.gotData)
		}

		var got domain.Manzana
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != "mz-1" || got.Number != "A" || !got.HasWater {
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
			{name: "not found", err: domain.ErrManzanaNotFound, wantStatus: http.StatusNotFound, wantCode: "manzana_not_found"},
			{name: "numero in use", err: domain.ErrManzanaNumberInUse, wantStatus: http.StatusConflict, wantCode: "manzana_numero_in_use"},
			{name: "missing numero", err: domain.ErrInvalidManzanaNumber, wantStatus: http.StatusBadRequest, wantCode: "invalid_manzana_numero"},
			{name: "unknown calle", err: domain.ErrUnknownCalle, wantStatus: http.StatusBadRequest, wantCode: "unknown_calle"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()
				updateManzana := &updateManzanaStub{err: test.err}
				recorder := performUpdateManzanaRequest(t, updateManzana, administradorVerifier(), "valid-token", manzanaRoute, manzanaDataBody())
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
		updateManzana := &updateManzanaStub{}
		recorder := performUpdateManzanaRequest(t, updateManzana, userVerifierStub{}, "", manzanaRoute, manzanaDataBody())
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if updateManzana.called {
			t.Error("use case should not be called without a token")
		}
	})
}
