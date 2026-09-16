package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/clients"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type deleteClientStub struct {
	err        error
	called     bool
	gotSubject string
	gotID      string
}

func (stub *deleteClientStub) Execute(_ context.Context, input clients.DeleteClientInput) error {
	stub.called = true
	stub.gotSubject = input.Subject
	stub.gotID = input.ID
	return stub.err
}

func performDeleteClientRequest(t *testing.T, deleteClient *deleteClientStub, verifier userVerifierStub, token, id string) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewDeleteClientHandler(deleteClient)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/v1/clientes/{id}", requireAuth(handler.Adapt(h, 5*time.Second)))

	return performRequest(t, mux, http.MethodDelete, "/api/v1/clientes/"+id, token, nil)
}

func TestDeleteClientRoute(t *testing.T) {
	t.Parallel()

	t.Run("gives the client de baja", func(t *testing.T) {
		t.Parallel()

		deleteClient := &deleteClientStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performDeleteClientRequest(t, deleteClient, verifier, "valid-token", validClientID)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
		}
		if deleteClient.gotID != validClientID {
			t.Errorf("id passed to use case = %q, want %q", deleteClient.gotID, validClientID)
		}
		if deleteClient.gotSubject != "user-1" {
			t.Errorf("subject passed to use case = %q, want %q", deleteClient.gotSubject, "user-1")
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		deleteClient := &deleteClientStub{}
		recorder := performDeleteClientRequest(t, deleteClient, userVerifierStub{}, "", validClientID)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if deleteClient.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects a non-uuid id before hitting the use case, without touching PostgreSQL", func(t *testing.T) {
		t.Parallel()

		deleteClient := &deleteClientStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performDeleteClientRequest(t, deleteClient, verifier, "valid-token", "not-a-uuid")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
		if deleteClient.called {
			t.Error("use case should not be called with a non-uuid id")
		}

		var got response.ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Code != "invalid_client_id" {
			t.Errorf("error code = %q, want %q", got.Code, "invalid_client_id")
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
			{name: "client not found", err: domain.ErrClienteNoEncontrado, wantStatus: http.StatusNotFound, wantCode: "client_not_found"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				deleteClient := &deleteClientStub{err: test.err}
				verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrativo}}}

				recorder := performDeleteClientRequest(t, deleteClient, verifier, "valid-token", validClientID)

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
}
