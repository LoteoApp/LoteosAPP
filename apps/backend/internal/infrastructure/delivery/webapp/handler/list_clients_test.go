package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type listClientsStub struct {
	clientes      []domain.Cliente
	err           error
	called        bool
	gotActorRoles []string
	gotSearch     string
}

func (stub *listClientsStub) Execute(_ context.Context, actorRoles []string, search string) ([]domain.Cliente, error) {
	stub.called = true
	stub.gotActorRoles = actorRoles
	stub.gotSearch = search
	return stub.clientes, stub.err
}

func performListClientsRequest(t *testing.T, listClients *listClientsStub, verifier userVerifierStub, token, query string) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewListClientsHandler(listClients)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/clientes", requireAuth(handler.Adapt(h, 5*time.Second)))

	path := "/api/v1/clientes"
	if query != "" {
		path += "?" + query
	}
	return performRequest(t, mux, http.MethodGet, path, token, nil)
}

func TestListClientsRoute(t *testing.T) {
	t.Parallel()

	t.Run("lists clients for an authorized role", func(t *testing.T) {
		t.Parallel()

		listClients := &listClientsStub{
			clientes: []domain.Cliente{{ID: "client-1", Nombre: "Ana", Apellido: "Perez", DNI: "30111222"}},
		}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrativo}}}

		recorder := performListClientsRequest(t, listClients, verifier, "valid-token", "q=perez")

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if listClients.gotSearch != "perez" {
			t.Errorf("search passed to use case = %q, want %q", listClients.gotSearch, "perez")
		}
		if len(listClients.gotActorRoles) != 1 || listClients.gotActorRoles[0] != domain.RolAdministrativo {
			t.Errorf("actor roles passed to use case = %v", listClients.gotActorRoles)
		}

		var got struct {
			Clientes []domain.Cliente `json:"clientes"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(got.Clientes) != 1 || got.Clientes[0].ID != "client-1" {
			t.Errorf("response = %#v", got)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		listClients := &listClientsStub{}
		recorder := performListClientsRequest(t, listClients, userVerifierStub{}, "", "")

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if listClients.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("propagates use case errors", func(t *testing.T) {
		t.Parallel()

		listClients := &listClientsStub{err: &domain.Error{Kind: domain.KindUnavailable, Code: "db_unavailable", Message: "no disponible"}}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performListClientsRequest(t, listClients, verifier, "valid-token", "")

		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
		}
	})

	t.Run("maps an unauthorized role to forbidden", func(t *testing.T) {
		t.Parallel()

		listClients := &listClientsStub{err: domain.ErrNoAutorizado}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolEscribano}}}

		recorder := performListClientsRequest(t, listClients, verifier, "valid-token", "")

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
		}
	})
}
