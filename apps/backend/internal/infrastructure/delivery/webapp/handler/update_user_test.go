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
	"loteosapp/backend/internal/business/usecase/users"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

const managedUserID = "0f8fad5b-d9cb-469f-a165-70867728950e"

type updateUserStub struct {
	usuario  domain.Usuario
	err      error
	called   bool
	gotInput users.UpdateUserInput
}

func (stub *updateUserStub) Execute(_ context.Context, input users.UpdateUserInput) (domain.Usuario, error) {
	stub.called = true
	stub.gotInput = input
	return stub.usuario, stub.err
}

func performUpdateUserRequest(t *testing.T, updateUser *updateUserStub, verifier userVerifierStub, token, id string, body any) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/v1/usuarios/{id}", middleware.RequireAuth(verifier)(
		handler.Adapt(handler.NewUpdateUserHandler(updateUser), 5*time.Second)))

	return performRequest(t, mux, http.MethodPatch, "/api/v1/usuarios/"+id, token, body)
}

func TestUpdateUserRoute(t *testing.T) {
	t.Parallel()

	t.Run("updates the user", func(t *testing.T) {
		t.Parallel()

		updateUser := &updateUserStub{usuario: domain.Usuario{
			ID: managedUserID, Nombre: "Ana María", Apellido: "Gómez", Rol: domain.RolEscribano,
		}}

		recorder := performUpdateUserRequest(t, updateUser, administradorVerifier(), "valid-token", managedUserID,
			map[string]string{"nombre": "Ana María"})

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}

		var got struct {
			ID     string `json:"id"`
			Nombre string `json:"nombre"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != managedUserID || got.Nombre != "Ana María" {
			t.Errorf("response = %#v", got)
		}
		if updateUser.gotInput.ID != managedUserID {
			t.Errorf("use case called with id %q, want %q", updateUser.gotInput.ID, managedUserID)
		}
		if updateUser.gotInput.Subject != "admin-1" {
			t.Errorf("use case called with subject %q, want %q", updateUser.gotInput.Subject, "admin-1")
		}
		if updateUser.gotInput.Nombre == nil || *updateUser.gotInput.Nombre != "Ana María" {
			t.Errorf("nombre passed to use case = %v", updateUser.gotInput.Nombre)
		}
		if updateUser.gotInput.Apellido != nil {
			t.Error("an omitted field should reach the use case as nil")
		}
	})

	t.Run("rejects an id that is not a uuid", func(t *testing.T) {
		t.Parallel()

		updateUser := &updateUserStub{}
		recorder := performUpdateUserRequest(t, updateUser, administradorVerifier(), "valid-token", "not-a-uuid",
			map[string]string{"nombre": "Ana"})

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}

		var got response.ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Code != "invalid_user_id" {
			t.Errorf("error code = %q, want %q", got.Code, "invalid_user_id")
		}
		if updateUser.called {
			t.Error("use case should not be called with a malformed id")
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		updateUser := &updateUserStub{}
		recorder := performUpdateUserRequest(t, updateUser, userVerifierStub{}, "", managedUserID,
			map[string]string{"nombre": "Ana"})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if updateUser.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		updateUser := &updateUserStub{}
		recorder := performUpdateUserRequest(t, updateUser, administradorVerifier(), "valid-token", managedUserID, "not-json")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if updateUser.called {
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
			{name: "invalid profile", err: domain.ErrPerfilInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_profile"},
			{name: "not found", err: domain.ErrUsuarioNoEncontrado, wantStatus: http.StatusNotFound, wantCode: "user_not_found"},
			{name: "actor not provisioned", err: domain.ErrActorNoAprovisionado, wantStatus: http.StatusForbidden, wantCode: "actor_not_provisioned"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				recorder := performUpdateUserRequest(t, &updateUserStub{err: test.err}, administradorVerifier(),
					"valid-token", managedUserID, map[string]string{"nombre": "Ana"})

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
