package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type updateLoteStub struct {
	lote       domain.Lote
	err        error
	called     bool
	gotActor   loteos.Actor
	gotLoteoID string
	gotLoteID  string
	gotData    domain.LoteData
}

func (stub *updateLoteStub) Execute(
	_ context.Context,
	actor loteos.Actor,
	loteoID, loteID string,
	data domain.LoteData,
) (domain.Lote, error) {
	stub.called = true
	stub.gotActor = actor
	stub.gotLoteoID = loteoID
	stub.gotLoteID = loteID
	stub.gotData = data
	return stub.lote, stub.err
}

func performUpdateLoteRequest(t *testing.T, updateLote *updateLoteStub, verifier userVerifierStub, token, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewUpdateLoteHandler(updateLote)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/v1/loteos/{loteoId}/lotes/{loteId}", requireAuth(handler.Adapt(h, 5*time.Second)))

	return performRequest(t, mux, http.MethodPatch, path, token, body)
}

const loteRoute = "/api/v1/loteos/loteo-1/lotes/lote-1"

func loteDataBody() map[string]any {
	return map[string]any{
		"numero":          "12",
		"precio":          150000.0,
		"moneda":          "ARS",
		"superficie":      300.5,
		"caracteristicas": "esquina",
	}
}

func TestUpdateLoteRoute(t *testing.T) {
	t.Parallel()

	t.Run("updates the lote and answers with it", func(t *testing.T) {
		t.Parallel()

		price := 150000.0
		updateLote := &updateLoteStub{lote: domain.Lote{
			ID: "lote-1", ManzanaID: "manzana-1", Number: "12", Price: &price, Currency: "ARS",
		}}

		recorder := performUpdateLoteRequest(t, updateLote, administradorVerifier(), "valid-token", loteRoute, loteDataBody())

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}

		var got domain.Lote
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != "lote-1" || got.Number != "12" {
			t.Errorf("response = %#v", got)
		}
		if got.Price == nil || *got.Price != 150000.0 {
			t.Errorf("price = %v, want 150000", got.Price)
		}
	})

	t.Run("takes both ids from the path", func(t *testing.T) {
		t.Parallel()

		updateLote := &updateLoteStub{}
		recorder := performUpdateLoteRequest(t, updateLote, administradorVerifier(), "valid-token", loteRoute, loteDataBody())

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		if updateLote.gotLoteoID != "loteo-1" || updateLote.gotLoteID != "lote-1" {
			t.Errorf("use case called with loteo %q and lote %q", updateLote.gotLoteoID, updateLote.gotLoteID)
		}
		if updateLote.gotActor.AuthProviderID != "admin-1" {
			t.Errorf("actor id = %q, want the token subject", updateLote.gotActor.AuthProviderID)
		}
	})

	t.Run("passes an omitted price and area as unset", func(t *testing.T) {
		t.Parallel()

		updateLote := &updateLoteStub{}
		recorder := performUpdateLoteRequest(t, updateLote, administradorVerifier(), "valid-token", loteRoute,
			map[string]any{"numero": "12"})

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		if updateLote.gotData.Price != nil || updateLote.gotData.Area != nil {
			t.Errorf("data = %#v, want price and area unset", updateLote.gotData)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		updateLote := &updateLoteStub{}
		recorder := performUpdateLoteRequest(t, updateLote, userVerifierStub{}, "", loteRoute, loteDataBody())

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if updateLote.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		updateLote := &updateLoteStub{}
		recorder := performUpdateLoteRequest(t, updateLote, administradorVerifier(), "valid-token", loteRoute, "not-json")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if updateLote.called {
			t.Error("use case should not be called with an invalid body")
		}
	})

	t.Run("rejects a body past the size limit", func(t *testing.T) {
		t.Parallel()

		// Valid JSON, but far larger than the values of a single lot: the
		// decoder must stop reading instead of allocating the whole body.
		oversized := `{"numero":"12","caracteristicas":"` + strings.Repeat("a", 64<<10) + `"}`

		updateLote := &updateLoteStub{}
		recorder := performUpdateLoteRequest(t, updateLote, administradorVerifier(), "valid-token", loteRoute, oversized)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if updateLote.called {
			t.Error("use case should not be called with an oversized body")
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
			{name: "lote not found", err: domain.ErrLoteNotFound, wantStatus: http.StatusNotFound, wantCode: "lote_not_found"},
			{name: "numero in use", err: domain.ErrLoteNumberInUse, wantStatus: http.StatusConflict, wantCode: "lote_numero_in_use"},
			{name: "missing numero", err: domain.ErrInvalidLoteNumber, wantStatus: http.StatusBadRequest, wantCode: "invalid_lote_numero"},
			{name: "invalid moneda", err: domain.ErrInvalidCurrency, wantStatus: http.StatusBadRequest, wantCode: "invalid_moneda"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				updateLote := &updateLoteStub{err: test.err}
				recorder := performUpdateLoteRequest(t, updateLote, administradorVerifier(), "valid-token", loteRoute, loteDataBody())

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
				if test.wantStatus == http.StatusInternalServerError && got.Message == test.err.Error() {
					t.Error("response exposes the internal error")
				}
			})
		}
	})

	t.Run("an agrimensor reaches the use case, which decides on the assignment", func(t *testing.T) {
		t.Parallel()

		updateLote := &updateLoteStub{}
		verifier := userVerifierStub{principal: supabase.Principal{
			Subject: "agrimensor-1",
			Roles:   []string{domain.RolAgrimensor},
		}}

		recorder := performUpdateLoteRequest(t, updateLote, verifier, "valid-token", loteRoute, loteDataBody())

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		if !domain.HasRole(updateLote.gotActor.Roles, domain.RolAgrimensor) {
			t.Errorf("actor roles = %v, want the token roles", updateLote.gotActor.Roles)
		}
	})
}
