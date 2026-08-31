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
	"loteosapp/backend/internal/business/usecase/clients"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type createClientStub struct {
	cliente       domain.Cliente
	err           error
	called        bool
	gotActorRoles []string
	gotSubject    string
	gotNombre     string
	gotApellido   string
	gotDNI        string
	gotCelular    *string
	gotEmail      *string
}

func (stub *createClientStub) Execute(_ context.Context, input clients.CreateClientInput) (domain.Cliente, error) {
	stub.called = true
	stub.gotActorRoles = input.ActorRoles
	stub.gotSubject = input.Subject
	stub.gotNombre = input.Nombre
	stub.gotApellido = input.Apellido
	stub.gotDNI = input.DNI
	stub.gotCelular = input.Celular
	stub.gotEmail = input.Email
	return stub.cliente, stub.err
}

func performCreateClientRequest(t *testing.T, createClient *createClientStub, verifier userVerifierStub, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewCreateClientHandler(createClient)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/clientes", requireAuth(handler.Adapt(h, 5*time.Second)))

	return performRequest(t, mux, http.MethodPost, "/api/v1/clientes", token, body)
}

func TestCreateClientRoute(t *testing.T) {
	t.Parallel()

	t.Run("creates client when actor has an allowed role", func(t *testing.T) {
		t.Parallel()

		createClient := &createClientStub{
			cliente: domain.Cliente{ID: "client-1", Nombre: "Ana", Apellido: "Perez", DNI: "30111222"},
		}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrativo}}}

		recorder := performCreateClientRequest(t, createClient, verifier, "valid-token",
			map[string]string{"nombre": "Ana", "apellido": "Perez", "dni": "30111222"})

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
		}

		var got struct {
			ID     string `json:"id"`
			Nombre string `json:"nombre"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != "client-1" || got.Nombre != "Ana" {
			t.Errorf("response = %#v", got)
		}
		if createClient.gotSubject != "user-1" {
			t.Errorf("subject passed to use case = %q, want %q", createClient.gotSubject, "user-1")
		}
		if len(createClient.gotActorRoles) != 1 || createClient.gotActorRoles[0] != domain.RolAdministrativo {
			t.Errorf("actor roles passed to use case = %v", createClient.gotActorRoles)
		}
		if createClient.gotDNI != "30111222" {
			t.Errorf("dni passed to use case = %q, want %q", createClient.gotDNI, "30111222")
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		createClient := &createClientStub{}
		recorder := performCreateClientRequest(t, createClient, userVerifierStub{}, "",
			map[string]string{"nombre": "Ana", "apellido": "Perez", "dni": "30111222"})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if createClient.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		createClient := &createClientStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performCreateClientRequest(t, createClient, verifier, "valid-token", "not-json")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if createClient.called {
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
			{name: "invalid client", err: domain.ErrClienteInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_client"},
			{name: "dni in use", err: domain.ErrDNIEnUso, wantStatus: http.StatusConflict, wantCode: "dni_in_use"},
			{name: "invalid email", err: domain.ErrEmailInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_email"},
			{name: "actor not provisioned", err: domain.ErrActorNoAprovisionado, wantStatus: http.StatusForbidden, wantCode: "actor_not_provisioned"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				createClient := &createClientStub{err: test.err}
				verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

				recorder := performCreateClientRequest(t, createClient, verifier, "valid-token",
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
