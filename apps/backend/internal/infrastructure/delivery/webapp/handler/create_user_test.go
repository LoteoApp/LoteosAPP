package handler_test

import (
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

type createUserStub struct {
	usuario       domain.Usuario
	tempPassword  string
	err           error
	called        bool
	gotActorRoles []string
	gotEmail      string
	gotRol        string
}

func (stub *createUserStub) Execute(_ context.Context, actorRoles []string, email, rol string) (domain.Usuario, string, error) {
	stub.called = true
	stub.gotActorRoles = actorRoles
	stub.gotEmail = email
	stub.gotRol = rol
	return stub.usuario, stub.tempPassword, stub.err
}

func performCreateUserRequest(t *testing.T, createUser *createUserStub, verifier userVerifierStub, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewCreateUserHandler(createUser)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/usuarios", requireAuth(http.HandlerFunc(h.Create)))

	return performRequest(t, mux, http.MethodPost, "/api/v1/usuarios", token, body)
}

func TestCreateUserRoute(t *testing.T) {
	t.Parallel()

	t.Run("creates user when actor is administrador", func(t *testing.T) {
		t.Parallel()

		createUser := &createUserStub{
			usuario:      domain.Usuario{ID: "u-1", Email: "ana@example.com", Rol: domain.RolAdministrativo},
			tempPassword: "temp-pass-123",
		}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "admin-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performCreateUserRequest(t, createUser, verifier, "valid-token",
			map[string]string{"email": "ana@example.com", "rol": domain.RolAdministrativo})

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
		}

		var got struct {
			Email             string `json:"email"`
			TemporaryPassword string `json:"temporaryPassword"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Email != "ana@example.com" || got.TemporaryPassword != "temp-pass-123" {
			t.Errorf("response = %#v", got)
		}
		if len(createUser.gotActorRoles) != 1 || createUser.gotActorRoles[0] != domain.RolAdministrador {
			t.Errorf("actor roles passed to use case = %v", createUser.gotActorRoles)
		}
		if createUser.gotEmail != "ana@example.com" || createUser.gotRol != domain.RolAdministrativo {
			t.Errorf("use case called with email=%q rol=%q", createUser.gotEmail, createUser.gotRol)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		createUser := &createUserStub{}
		recorder := performCreateUserRequest(t, createUser, userVerifierStub{}, "",
			map[string]string{"email": "ana@example.com", "rol": domain.RolAdministrativo})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if createUser.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		createUser := &createUserStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "admin-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performCreateUserRequest(t, createUser, verifier, "valid-token", "not-json")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if createUser.called {
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
			{name: "invalid email", err: domain.ErrEmailInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_email"},
			{name: "invalid rol", err: domain.ErrRolInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_rol"},
			{name: "email in use", err: domain.ErrEmailEnUso, wantStatus: http.StatusConflict, wantCode: "email_in_use"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				createUser := &createUserStub{err: test.err}
				verifier := userVerifierStub{principal: supabase.Principal{Subject: "admin-1", Roles: []string{domain.RolAdministrador}}}

				recorder := performCreateUserRequest(t, createUser, verifier, "valid-token",
					map[string]string{"email": "ana@example.com", "rol": domain.RolAdministrativo})

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
