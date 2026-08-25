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
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type updateClientStub struct {
	cliente     domain.Cliente
	err         error
	called      bool
	gotSubject  string
	gotID       string
	gotNombre   string
	gotApellido string
	gotDNI      string
}

func (stub *updateClientStub) Execute(
	_ context.Context,
	_ []string,
	subject, id, nombre, apellido, dni string,
	_, _ *string,
) (domain.Cliente, error) {
	stub.called = true
	stub.gotSubject = subject
	stub.gotID = id
	stub.gotNombre = nombre
	stub.gotApellido = apellido
	stub.gotDNI = dni
	return stub.cliente, stub.err
}

func performUpdateClientRequest(t *testing.T, updateClient *updateClientStub, verifier userVerifierStub, token, id string, body any) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewUpdateClientHandler(updateClient)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/v1/clientes/{id}", requireAuth(handler.Adapt(h, 5*time.Second)))

	return performRequest(t, mux, http.MethodPatch, "/api/v1/clientes/"+id, token, body)
}

func TestUpdateClientRoute(t *testing.T) {
	t.Parallel()

	t.Run("updates client and passes the path id", func(t *testing.T) {
		t.Parallel()

		updateClient := &updateClientStub{
			cliente: domain.Cliente{ID: "client-1", Nombre: "Ana", Apellido: "Perez", DNI: "30111222"},
		}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolInmobiliaria}}}

		recorder := performUpdateClientRequest(t, updateClient, verifier, "valid-token", "client-1",
			map[string]string{"nombre": "Ana", "apellido": "Perez", "dni": "30111222"})

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if updateClient.gotID != "client-1" {
			t.Errorf("id passed to use case = %q, want %q", updateClient.gotID, "client-1")
		}
		if updateClient.gotSubject != "user-1" {
			t.Errorf("subject passed to use case = %q, want %q", updateClient.gotSubject, "user-1")
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		updateClient := &updateClientStub{}
		recorder := performUpdateClientRequest(t, updateClient, userVerifierStub{}, "", "client-1",
			map[string]string{"nombre": "Ana", "apellido": "Perez", "dni": "30111222"})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if updateClient.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		updateClient := &updateClientStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performUpdateClientRequest(t, updateClient, verifier, "valid-token", "client-1", "not-json")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if updateClient.called {
			t.Error("use case should not be called with an invalid body")
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
			{name: "dni in use", err: domain.ErrDNIEnUso, wantStatus: http.StatusConflict, wantCode: "dni_in_use"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				updateClient := &updateClientStub{err: test.err}
				verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

				recorder := performUpdateClientRequest(t, updateClient, verifier, "valid-token", "client-1",
					map[string]string{"nombre": "Ana", "apellido": "Perez", "dni": "30111222"})

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
