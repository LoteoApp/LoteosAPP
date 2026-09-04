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
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type reactivateUserStub struct {
	usuario    domain.Usuario
	err        error
	called     bool
	gotSubject string
	gotID      string
}

func (stub *reactivateUserStub) Execute(_ context.Context, _ []string, subject, id string) (domain.Usuario, error) {
	stub.called = true
	stub.gotSubject = subject
	stub.gotID = id
	return stub.usuario, stub.err
}

func performReactivateUserRequest(t *testing.T, reactivateUser *reactivateUserStub, verifier userVerifierStub, token, id string) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/usuarios/{id}/reactivar", middleware.RequireAuth(verifier)(
		handler.Adapt(handler.NewReactivateUserHandler(reactivateUser), 5*time.Second)))

	return performRequest(t, mux, http.MethodPost, "/api/v1/usuarios/"+id+"/reactivar", token, nil)
}

func TestReactivateUserRoute(t *testing.T) {
	t.Parallel()

	t.Run("reactivates the user", func(t *testing.T) {
		t.Parallel()

		reactivateUser := &reactivateUserStub{usuario: domain.Usuario{
			ID: managedUserID, Nombre: "Ana", Apellido: "Gómez", Rol: domain.RolEscribano,
		}}

		recorder := performReactivateUserRequest(t, reactivateUser, administradorVerifier(), "valid-token", managedUserID)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}

		var got struct {
			ID        string     `json:"id"`
			FechaBaja *time.Time `json:"fechaBaja"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != managedUserID {
			t.Errorf("response id = %q, want %q", got.ID, managedUserID)
		}
		if got.FechaBaja != nil {
			t.Error("response should carry a nil fechaBaja")
		}
		if reactivateUser.gotID != managedUserID || reactivateUser.gotSubject != "admin-1" {
			t.Errorf("use case called with id %q subject %q", reactivateUser.gotID, reactivateUser.gotSubject)
		}
	})

	t.Run("rejects an id that is not a uuid", func(t *testing.T) {
		t.Parallel()

		reactivateUser := &reactivateUserStub{}
		recorder := performReactivateUserRequest(t, reactivateUser, administradorVerifier(), "valid-token", "not-a-uuid")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if reactivateUser.called {
			t.Error("use case should not be called with a malformed id")
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		reactivateUser := &reactivateUserStub{}
		recorder := performReactivateUserRequest(t, reactivateUser, userVerifierStub{}, "", managedUserID)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if reactivateUser.called {
			t.Error("use case should not be called without a token")
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
			{name: "not found", err: domain.ErrUsuarioNoEncontrado, wantStatus: http.StatusNotFound, wantCode: "user_not_found"},
			{name: "already active", err: domain.ErrUsuarioYaActivo, wantStatus: http.StatusConflict, wantCode: "user_already_active"},
			{name: "actor not provisioned", err: domain.ErrActorNoAprovisionado, wantStatus: http.StatusForbidden, wantCode: "actor_not_provisioned"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				recorder := performReactivateUserRequest(t, &reactivateUserStub{err: test.err},
					administradorVerifier(), "valid-token", managedUserID)

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
