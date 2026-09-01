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
)

type listLoteosStub struct {
	result   []domain.LoteoSummary
	err      error
	called   bool
	gotActor loteos.Actor
	gotInput loteos.ListLoteosInput
}

func (stub *listLoteosStub) Execute(_ context.Context, input loteos.ListLoteosInput) ([]domain.LoteoSummary, error) {
	stub.called = true
	stub.gotActor = input.Actor
	stub.gotInput = input
	return stub.result, stub.err
}

func performListLoteosRequest(t *testing.T, listLoteos *listLoteosStub, verifier userVerifierStub, token, query string) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewListLoteosHandler(listLoteos)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/loteos", requireAuth(handler.Adapt(h, 5*time.Second)))

	path := "/api/v1/loteos"
	if query != "" {
		path += "?" + query
	}
	return performRequest(t, mux, http.MethodGet, path, token, nil)
}

func TestListLoteosRoute(t *testing.T) {
	t.Parallel()

	t.Run("lists loteos for an authorized caller", func(t *testing.T) {
		t.Parallel()

		listLoteos := &listLoteosStub{result: []domain.LoteoSummary{
			{ID: "loteo-1", Name: "Norte", ManzanaCount: 3, LoteCount: 40, HasPlan: true},
		}}
		verifier := userVerifierStub{principal: supabase.Principal{
			Subject: "user-1", Roles: []string{domain.RolAgrimensor},
		}}

		recorder := performListLoteosRequest(t, listLoteos, verifier, "valid-token", "q=norte")

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if listLoteos.gotInput.Search != "norte" {
			t.Errorf("search passed to use case = %q, want %q", listLoteos.gotInput.Search, "norte")
		}
		if listLoteos.gotActor.AuthProviderID != "user-1" || !domain.HasRole(listLoteos.gotActor.Roles, domain.RolAgrimensor) {
			t.Errorf("actor passed to use case = %#v", listLoteos.gotActor)
		}

		var got struct {
			Loteos []domain.LoteoSummary `json:"loteos"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(got.Loteos) != 1 || got.Loteos[0].ID != "loteo-1" || got.Loteos[0].ManzanaCount != 3 {
			t.Errorf("response = %#v", got.Loteos)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		listLoteos := &listLoteosStub{}
		recorder := performListLoteosRequest(t, listLoteos, userVerifierStub{}, "", "")

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if listLoteos.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("maps an unauthorized role to forbidden", func(t *testing.T) {
		t.Parallel()

		listLoteos := &listLoteosStub{err: domain.ErrNoAutorizado}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{"other"}}}

		recorder := performListLoteosRequest(t, listLoteos, verifier, "valid-token", "")

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
		}
	})

	t.Run("hides an unexpected error behind a 500", func(t *testing.T) {
		t.Parallel()

		listLoteos := &listLoteosStub{err: domain.ErrDatabaseUnavailable.WithCause(context.DeadlineExceeded)}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performListLoteosRequest(t, listLoteos, verifier, "valid-token", "")

		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
		}
	})
}
