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

type completeProfileStub struct {
	usuario     domain.Usuario
	err         error
	called      bool
	gotSubject  string
	gotNombre   string
	gotApellido string
}

func (stub *completeProfileStub) Execute(_ context.Context, subject, nombre, apellido string) (domain.Usuario, error) {
	stub.called = true
	stub.gotSubject = subject
	stub.gotNombre = nombre
	stub.gotApellido = apellido
	return stub.usuario, stub.err
}

func performCompleteProfileRequest(t *testing.T, completeProfile *completeProfileStub, verifier userVerifierStub, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewCompleteProfileHandler(completeProfile)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/v1/usuarios/me", requireAuth(handler.Adapt(h, 5*time.Second)))

	return performRequest(t, mux, http.MethodPatch, "/api/v1/usuarios/me", token, body)
}

func TestCompleteProfileRoute(t *testing.T) {
	t.Parallel()

	t.Run("completes the caller's own profile", func(t *testing.T) {
		t.Parallel()

		completeProfile := &completeProfileStub{
			usuario: domain.Usuario{ID: "u-1", Nombre: "Ana", Apellido: "Gómez", PerfilCompleto: true},
		}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAgrimensor}}}

		recorder := performCompleteProfileRequest(t, completeProfile, verifier, "valid-token",
			map[string]string{"nombre": "Ana", "apellido": "Gómez"})

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if completeProfile.gotSubject != "user-1" {
			t.Errorf("subject passed to use case = %q, want %q", completeProfile.gotSubject, "user-1")
		}
		if completeProfile.gotNombre != "Ana" || completeProfile.gotApellido != "Gómez" {
			t.Errorf("use case called with nombre=%q apellido=%q", completeProfile.gotNombre, completeProfile.gotApellido)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		completeProfile := &completeProfileStub{}
		recorder := performCompleteProfileRequest(t, completeProfile, userVerifierStub{}, "",
			map[string]string{"nombre": "Ana", "apellido": "Gómez"})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if completeProfile.called {
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
			{name: "invalid profile", err: domain.ErrPerfilInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_profile"},
			{name: "user not found", err: domain.ErrUsuarioNoEncontrado, wantStatus: http.StatusNotFound, wantCode: "user_not_found"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				completeProfile := &completeProfileStub{err: test.err}
				verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1"}}

				recorder := performCompleteProfileRequest(t, completeProfile, verifier, "valid-token",
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
