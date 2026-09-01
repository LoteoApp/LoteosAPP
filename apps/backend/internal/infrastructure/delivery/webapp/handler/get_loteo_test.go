package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type getLoteoStub struct {
	loteo      domain.Loteo
	err        error
	called     bool
	gotActor   loteos.Actor
	gotLoteoID string
}

func (stub *getLoteoStub) Execute(_ context.Context, actor loteos.Actor, loteoID string) (domain.Loteo, error) {
	stub.called = true
	stub.gotActor = actor
	stub.gotLoteoID = loteoID
	return stub.loteo, stub.err
}

func performGetLoteoRequest(t *testing.T, getLoteo *getLoteoStub, verifier userVerifierStub, token, loteoID string) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewGetLoteoHandler(getLoteo)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/loteos/{loteoId}", requireAuth(handler.Adapt(h, 5*time.Second)))

	return performRequest(t, mux, http.MethodGet, "/api/v1/loteos/"+loteoID, token, nil)
}

func TestGetLoteoRoute(t *testing.T) {
	t.Parallel()

	t.Run("returns the loteo with its plan", func(t *testing.T) {
		t.Parallel()

		price := 150000.0
		getLoteo := &getLoteoStub{loteo: domain.Loteo{
			ID:       "loteo-1",
			Name:     "Norte",
			Boundary: domain.Polygon{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}},
			Manzanas: []domain.Manzana{{ID: "manzana-1", Number: "1"}},
			Lotes:    []domain.Lote{{ID: "lote-1", ManzanaID: "manzana-1", Number: "7", Price: &price}},
			Calles:   []domain.Calle{{ID: "calle-1", Name: "Los Álamos"}},
		}}
		verifier := userVerifierStub{principal: supabase.Principal{
			Subject: "user-1", Roles: []string{domain.RolInmobiliaria},
		}}

		recorder := performGetLoteoRequest(t, getLoteo, verifier, "valid-token", "loteo-1")

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if getLoteo.gotLoteoID != "loteo-1" {
			t.Errorf("loteo id passed to use case = %q", getLoteo.gotLoteoID)
		}
		if getLoteo.gotActor.AuthProviderID != "user-1" {
			t.Errorf("actor passed to use case = %#v", getLoteo.gotActor)
		}

		var got domain.Loteo
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != "loteo-1" || len(got.Boundary) != 3 || len(got.Lotes) != 1 || got.Lotes[0].ManzanaID != "manzana-1" {
			t.Errorf("response = %#v", got)
		}
	})

	t.Run("maps an unknown loteo to a 404", func(t *testing.T) {
		t.Parallel()

		getLoteo := &getLoteoStub{err: domain.ErrLoteoNotFound}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performGetLoteoRequest(t, getLoteo, verifier, "valid-token", "missing")

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
		}
		var got response.ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Code != "loteo_not_found" {
			t.Errorf("error code = %q, want loteo_not_found", got.Code)
		}
	})

	t.Run("maps an unauthorized role to forbidden", func(t *testing.T) {
		t.Parallel()

		getLoteo := &getLoteoStub{err: domain.ErrNoAutorizado}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{"other"}}}

		recorder := performGetLoteoRequest(t, getLoteo, verifier, "valid-token", "loteo-1")

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		getLoteo := &getLoteoStub{}
		recorder := performGetLoteoRequest(t, getLoteo, userVerifierStub{}, "", "loteo-1")

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if getLoteo.called {
			t.Error("use case should not be called without a token")
		}
	})
}
