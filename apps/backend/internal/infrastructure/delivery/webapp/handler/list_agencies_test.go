package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/agencies"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type listAgenciesStub struct {
	agencies      []domain.Agency
	err           error
	called        bool
	gotActorRoles []string
	gotSearch     string
}

func (stub *listAgenciesStub) Execute(_ context.Context, input agencies.ListAgenciesInput) ([]domain.Agency, error) {
	stub.called = true
	stub.gotActorRoles = input.ActorRoles
	stub.gotSearch = input.Search
	return stub.agencies, stub.err
}

func performListAgenciesRequest(t *testing.T, listAgencies *listAgenciesStub, verifier userVerifierStub, token, query string) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewListAgenciesHandler(listAgencies)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/inmobiliarias", requireAuth(handler.Adapt(h, 5*time.Second)))

	path := "/api/v1/inmobiliarias"
	if query != "" {
		path += "?" + query
	}
	return performRequest(t, mux, http.MethodGet, path, token, nil)
}

func TestListAgenciesRoute(t *testing.T) {
	t.Parallel()

	t.Run("lists agencies for an authorized role", func(t *testing.T) {
		t.Parallel()

		listAgencies := &listAgenciesStub{
			agencies: []domain.Agency{{ID: validAgencyID, BusinessName: "Lotes del Sur"}},
		}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrativo}}}

		recorder := performListAgenciesRequest(t, listAgencies, verifier, "valid-token", "q=sur")

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if listAgencies.gotSearch != "sur" {
			t.Errorf("search passed to use case = %q, want %q", listAgencies.gotSearch, "sur")
		}
		if len(listAgencies.gotActorRoles) != 1 || listAgencies.gotActorRoles[0] != domain.RolAdministrativo {
			t.Errorf("actor roles passed to use case = %v", listAgencies.gotActorRoles)
		}

		var got struct {
			Agencies []domain.Agency `json:"inmobiliarias"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(got.Agencies) != 1 || got.Agencies[0].ID != validAgencyID {
			t.Errorf("response = %#v", got)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		listAgencies := &listAgenciesStub{}
		recorder := performListAgenciesRequest(t, listAgencies, userVerifierStub{}, "", "")

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if listAgencies.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("propagates use case errors", func(t *testing.T) {
		t.Parallel()

		listAgencies := &listAgenciesStub{err: &domain.Error{Kind: domain.KindUnavailable, Code: "db_unavailable", Message: "no disponible"}}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performListAgenciesRequest(t, listAgencies, verifier, "valid-token", "")

		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("maps an unauthorized role to forbidden", func(t *testing.T) {
		t.Parallel()

		listAgencies := &listAgenciesStub{err: domain.ErrNoAutorizado}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolEscribano}}}

		recorder := performListAgenciesRequest(t, listAgencies, verifier, "valid-token", "")

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
		}
	})
}
