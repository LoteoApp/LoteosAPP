package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type clientServiceStub struct {
	listResult []domain.Cliente
	listErr    error
	listCalled bool
	gotBuscar  string

	createResult  domain.Cliente
	createErr     error
	createCalled  bool
	gotCreated    domain.Cliente
	gotActorRoles []string
	gotSubject    string

	updateResult domain.Cliente
	updateErr    error
	updateCalled bool
	gotUpdated   domain.Cliente

	deleteErr    error
	deleteCalled bool
	gotDeletedID string
}

func (stub *clientServiceStub) List(_ context.Context, actorRoles []string, buscar string) ([]domain.Cliente, error) {
	stub.listCalled = true
	stub.gotActorRoles = actorRoles
	stub.gotBuscar = buscar
	return stub.listResult, stub.listErr
}

func (stub *clientServiceStub) Create(_ context.Context, actorRoles []string, actorSubject string, cliente domain.Cliente) (domain.Cliente, error) {
	stub.createCalled = true
	stub.gotActorRoles = actorRoles
	stub.gotSubject = actorSubject
	stub.gotCreated = cliente
	return stub.createResult, stub.createErr
}

func (stub *clientServiceStub) Update(_ context.Context, actorRoles []string, actorSubject string, cliente domain.Cliente) (domain.Cliente, error) {
	stub.updateCalled = true
	stub.gotActorRoles = actorRoles
	stub.gotSubject = actorSubject
	stub.gotUpdated = cliente
	return stub.updateResult, stub.updateErr
}

func (stub *clientServiceStub) Delete(_ context.Context, actorRoles []string, actorSubject, id string) error {
	stub.deleteCalled = true
	stub.gotActorRoles = actorRoles
	stub.gotSubject = actorSubject
	stub.gotDeletedID = id
	return stub.deleteErr
}

func performClientRequest(t *testing.T, service *clientServiceStub, verifier userVerifierStub, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	ch := handler.NewClientHandler(service)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/clientes", requireAuth(http.HandlerFunc(ch.List)))
	mux.Handle("POST /api/v1/clientes", requireAuth(http.HandlerFunc(ch.Create)))
	mux.Handle("PATCH /api/v1/clientes/{id}", requireAuth(http.HandlerFunc(ch.Update)))
	mux.Handle("DELETE /api/v1/clientes/{id}", requireAuth(http.HandlerFunc(ch.Delete)))

	var reader *bytes.Reader
	switch v := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(v))
	default:
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}

	request := httptest.NewRequest(method, path, reader)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	return recorder
}

func administrativoVerifier() userVerifierStub {
	return userVerifierStub{principal: supabase.Principal{Subject: "sb-123", Roles: []string{domain.RolAdministrativo}}}
}

func TestClientHandlerList(t *testing.T) {
	t.Parallel()

	t.Run("returns the clients and forwards the search term", func(t *testing.T) {
		t.Parallel()

		service := &clientServiceStub{listResult: []domain.Cliente{{ID: "c-1", Nombre: "Ana", Apellido: "Pérez"}}}

		recorder := performClientRequest(t, service, administrativoVerifier(), http.MethodGet, "/api/v1/clientes?buscar=ana", "valid-token", nil)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}

		var got []domain.Cliente
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(got) != 1 || got[0].Nombre != "Ana" {
			t.Errorf("response = %#v", got)
		}
		if service.gotBuscar != "ana" {
			t.Errorf("search term passed to service = %q, want %q", service.gotBuscar, "ana")
		}
	})

	t.Run("returns an empty array when there are no clients", func(t *testing.T) {
		t.Parallel()

		service := &clientServiceStub{listResult: []domain.Cliente{}}

		recorder := performClientRequest(t, service, administrativoVerifier(), http.MethodGet, "/api/v1/clientes", "valid-token", nil)

		if body := recorder.Body.String(); body != "[]\n" && body != "[]" {
			t.Errorf("body = %q, want an empty JSON array", body)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		service := &clientServiceStub{}
		recorder := performClientRequest(t, service, userVerifierStub{}, http.MethodGet, "/api/v1/clientes", "", nil)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if service.listCalled {
			t.Error("service should not be called without a token")
		}
	})
}

func TestClientHandlerCreate(t *testing.T) {
	t.Parallel()

	t.Run("creates a client", func(t *testing.T) {
		t.Parallel()

		service := &clientServiceStub{createResult: domain.Cliente{ID: "c-1", Nombre: "Ana", Apellido: "Pérez", DNI: "30111222"}}

		recorder := performClientRequest(t, service, administrativoVerifier(), http.MethodPost, "/api/v1/clientes", "valid-token",
			map[string]string{"nombre": "Ana", "apellido": "Pérez", "dni": "30111222", "celular": "1122334455", "email": "ana@example.com"})

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
		}
		if service.gotCreated.Nombre != "Ana" || service.gotCreated.DNI != "30111222" {
			t.Errorf("service called with %#v", service.gotCreated)
		}
		if service.gotCreated.Celular != "1122334455" || service.gotCreated.Email != "ana@example.com" {
			t.Errorf("service called with contact %#v", service.gotCreated)
		}
		if service.gotSubject != "sb-123" {
			t.Errorf("subject passed to service = %q, want %q", service.gotSubject, "sb-123")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		service := &clientServiceStub{}

		recorder := performClientRequest(t, service, administrativoVerifier(), http.MethodPost, "/api/v1/clientes", "valid-token", "not-json")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if service.createCalled {
			t.Error("service should not be called with an invalid body")
		}
	})

	t.Run("maps service errors to the right status", func(t *testing.T) {
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
			{name: "actor not found", err: domain.ErrUsuarioNoEncontrado, wantStatus: http.StatusNotFound, wantCode: "user_not_found"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				service := &clientServiceStub{createErr: test.err}

				recorder := performClientRequest(t, service, administrativoVerifier(), http.MethodPost, "/api/v1/clientes", "valid-token",
					map[string]string{"nombre": "Ana", "apellido": "Pérez", "dni": "30111222"})

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
}

func TestClientHandlerUpdate(t *testing.T) {
	t.Parallel()

	t.Run("updates the client named in the path", func(t *testing.T) {
		t.Parallel()

		service := &clientServiceStub{updateResult: domain.Cliente{ID: "c-1", Nombre: "Ana María"}}

		recorder := performClientRequest(t, service, administrativoVerifier(), http.MethodPatch, "/api/v1/clientes/c-1", "valid-token",
			map[string]string{"nombre": "Ana María", "apellido": "Pérez", "dni": "30111222"})

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if service.gotUpdated.ID != "c-1" {
			t.Errorf("service called with id = %q, want %q", service.gotUpdated.ID, "c-1")
		}
		if service.gotUpdated.Nombre != "Ana María" {
			t.Errorf("service called with %#v", service.gotUpdated)
		}
	})

	t.Run("maps a missing client to 404", func(t *testing.T) {
		t.Parallel()

		service := &clientServiceStub{updateErr: domain.ErrClienteNoEncontrado}

		recorder := performClientRequest(t, service, administrativoVerifier(), http.MethodPatch, "/api/v1/clientes/c-1", "valid-token",
			map[string]string{"nombre": "Ana", "apellido": "Pérez", "dni": "30111222"})

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
		}

		var got response.ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Code != "client_not_found" {
			t.Errorf("error code = %q, want %q", got.Code, "client_not_found")
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		service := &clientServiceStub{}
		recorder := performClientRequest(t, service, userVerifierStub{}, http.MethodPatch, "/api/v1/clientes/c-1", "",
			map[string]string{"nombre": "Ana"})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if service.updateCalled {
			t.Error("service should not be called without a token")
		}
	})
}

func TestClientHandlerDelete(t *testing.T) {
	t.Parallel()

	t.Run("gives the baja and returns no content", func(t *testing.T) {
		t.Parallel()

		service := &clientServiceStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "sb-123", Roles: []string{domain.RolAdministrador}}}

		recorder := performClientRequest(t, service, verifier, http.MethodDelete, "/api/v1/clientes/c-1", "valid-token", nil)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
		}
		if service.gotDeletedID != "c-1" {
			t.Errorf("service called with id = %q, want %q", service.gotDeletedID, "c-1")
		}
		if recorder.Body.Len() != 0 {
			t.Errorf("body = %q, want empty", recorder.Body.String())
		}
	})

	t.Run("returns 403 when the caller is not administrador", func(t *testing.T) {
		t.Parallel()

		service := &clientServiceStub{deleteErr: domain.ErrNoAutorizado}

		recorder := performClientRequest(t, service, administrativoVerifier(), http.MethodDelete, "/api/v1/clientes/c-1", "valid-token", nil)

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
		}

		var got response.ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Code != "forbidden" {
			t.Errorf("error code = %q, want %q", got.Code, "forbidden")
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		service := &clientServiceStub{}
		recorder := performClientRequest(t, service, userVerifierStub{}, http.MethodDelete, "/api/v1/clientes/c-1", "", nil)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if service.deleteCalled {
			t.Error("service should not be called without a token")
		}
	})
}
