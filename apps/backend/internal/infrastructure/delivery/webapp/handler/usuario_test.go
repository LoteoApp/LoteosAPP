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

type userServiceStub struct {
	createUsuario      domain.Usuario
	createTempPassword string
	createErr          error
	createCalled       bool
	gotActorRoles      []string
	gotEmail, gotRol   string

	completeUsuario                    domain.Usuario
	completeErr                        error
	completeCalled                     bool
	gotSubject, gotNombre, gotApellido string
}

func (stub *userServiceStub) CreateUser(_ context.Context, actorRoles []string, email, rol string) (domain.Usuario, string, error) {
	stub.createCalled = true
	stub.gotActorRoles = actorRoles
	stub.gotEmail = email
	stub.gotRol = rol
	return stub.createUsuario, stub.createTempPassword, stub.createErr
}

func (stub *userServiceStub) CompleteProfile(_ context.Context, subject, nombre, apellido string) (domain.Usuario, error) {
	stub.completeCalled = true
	stub.gotSubject = subject
	stub.gotNombre = nombre
	stub.gotApellido = apellido
	return stub.completeUsuario, stub.completeErr
}

type userVerifierStub struct {
	principal supabase.Principal
	err       error
}

func (stub userVerifierStub) Verify(context.Context, string) (supabase.Principal, error) {
	return stub.principal, stub.err
}

func performUserRequest(t *testing.T, service *userServiceStub, verifier userVerifierStub, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	uh := handler.NewUserHandler(service)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/usuarios", requireAuth(http.HandlerFunc(uh.Create)))
	mux.Handle("PATCH /api/v1/usuarios/me", requireAuth(http.HandlerFunc(uh.CompleteProfile)))

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

func TestUserHandlerCreate(t *testing.T) {
	t.Parallel()

	t.Run("creates user when actor is administrador", func(t *testing.T) {
		t.Parallel()

		service := &userServiceStub{
			createUsuario:      domain.Usuario{ID: "u-1", Email: "ana@example.com", Rol: domain.RolAdministrativo},
			createTempPassword: "temp-pass-123",
		}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "admin-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performUserRequest(t, service, verifier, http.MethodPost, "/api/v1/usuarios", "valid-token",
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
		if len(service.gotActorRoles) != 1 || service.gotActorRoles[0] != domain.RolAdministrador {
			t.Errorf("actor roles passed to service = %v", service.gotActorRoles)
		}
		if service.gotEmail != "ana@example.com" || service.gotRol != domain.RolAdministrativo {
			t.Errorf("service called with email=%q rol=%q", service.gotEmail, service.gotRol)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		service := &userServiceStub{}
		recorder := performUserRequest(t, service, userVerifierStub{}, http.MethodPost, "/api/v1/usuarios", "",
			map[string]string{"email": "ana@example.com", "rol": domain.RolAdministrativo})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if service.createCalled {
			t.Error("service should not be called without a token")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		service := &userServiceStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "admin-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performUserRequest(t, service, verifier, http.MethodPost, "/api/v1/usuarios", "valid-token", "not-json")

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
			{name: "invalid email", err: domain.ErrEmailInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_email"},
			{name: "invalid rol", err: domain.ErrRolInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_rol"},
			{name: "email in use", err: domain.ErrEmailEnUso, wantStatus: http.StatusConflict, wantCode: "email_in_use"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				service := &userServiceStub{createErr: test.err}
				verifier := userVerifierStub{principal: supabase.Principal{Subject: "admin-1", Roles: []string{domain.RolAdministrador}}}

				recorder := performUserRequest(t, service, verifier, http.MethodPost, "/api/v1/usuarios", "valid-token",
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

func TestUserHandlerCompleteProfile(t *testing.T) {
	t.Parallel()

	t.Run("completes the caller's own profile", func(t *testing.T) {
		t.Parallel()

		service := &userServiceStub{
			completeUsuario: domain.Usuario{ID: "u-1", Nombre: "Ana", Apellido: "Gómez", PerfilCompleto: true},
		}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAgrimensor}}}

		recorder := performUserRequest(t, service, verifier, http.MethodPatch, "/api/v1/usuarios/me", "valid-token",
			map[string]string{"nombre": "Ana", "apellido": "Gómez"})

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if service.gotSubject != "user-1" {
			t.Errorf("subject passed to service = %q, want %q", service.gotSubject, "user-1")
		}
		if service.gotNombre != "Ana" || service.gotApellido != "Gómez" {
			t.Errorf("service called with nombre=%q apellido=%q", service.gotNombre, service.gotApellido)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		service := &userServiceStub{}
		recorder := performUserRequest(t, service, userVerifierStub{}, http.MethodPatch, "/api/v1/usuarios/me", "",
			map[string]string{"nombre": "Ana", "apellido": "Gómez"})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if service.completeCalled {
			t.Error("service should not be called without a token")
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
			{name: "invalid profile", err: domain.ErrPerfilInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_profile"},
			{name: "user not found", err: domain.ErrUsuarioNoEncontrado, wantStatus: http.StatusNotFound, wantCode: "user_not_found"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				service := &userServiceStub{completeErr: test.err}
				verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1"}}

				recorder := performUserRequest(t, service, verifier, http.MethodPatch, "/api/v1/usuarios/me", "valid-token",
					map[string]string{"nombre": "Ana", "apellido": "Gómez"})

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
