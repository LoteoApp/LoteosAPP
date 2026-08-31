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

type createSurveyorStub struct {
	usuario      domain.Usuario
	tempPassword string
	err          error
	called       bool
	gotRoles     []string
	gotNombre    string
	gotApellido  string
	gotEmail     string
}

func (stub *createSurveyorStub) Execute(_ context.Context, actorRoles []string, nombre, apellido, email string) (domain.Usuario, string, error) {
	stub.called = true
	stub.gotRoles = actorRoles
	stub.gotNombre = nombre
	stub.gotApellido = apellido
	stub.gotEmail = email
	return stub.usuario, stub.tempPassword, stub.err
}

func performCreateSurveyorRequest(t *testing.T, createSurveyor *createSurveyorStub, verifier userVerifierStub, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/agrimensores", middleware.RequireAuth(verifier)(
		handler.Adapt(handler.NewCreateSurveyorHandler(createSurveyor), 5*time.Second)))

	return performRequest(t, mux, http.MethodPost, "/api/v1/agrimensores", token, body)
}

func TestCreateSurveyorRoute(t *testing.T) {
	t.Parallel()

	t.Run("creates the agrimensor when actor is administrador", func(t *testing.T) {
		t.Parallel()

		createSurveyor := &createSurveyorStub{
			usuario: domain.Usuario{
				ID: "agri-1", Email: "ana@example.com", Nombre: "Ana",
				Apellido: "Gómez", Rol: domain.RolAgrimensor,
			},
			tempPassword: "temp-pass-123",
		}

		recorder := performCreateSurveyorRequest(t, createSurveyor, administradorVerifier(), "valid-token",
			map[string]string{"nombre": "Ana", "apellido": "Gómez", "email": "ana@example.com"})

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
		}

		var got struct {
			ID                string  `json:"id"`
			Email             string  `json:"email"`
			Rol               string  `json:"rol"`
			FechaBaja         *string `json:"fechaBaja"`
			TemporaryPassword string  `json:"temporaryPassword"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != "agri-1" || got.Email != "ana@example.com" || got.Rol != domain.RolAgrimensor {
			t.Errorf("response = %#v", got)
		}
		if got.FechaBaja != nil {
			t.Errorf("fechaBaja = %v, want null for an active agrimensor", *got.FechaBaja)
		}
		if got.TemporaryPassword != "temp-pass-123" {
			t.Errorf("temporaryPassword = %q, want %q", got.TemporaryPassword, "temp-pass-123")
		}
		if createSurveyor.gotNombre != "Ana" || createSurveyor.gotApellido != "Gómez" || createSurveyor.gotEmail != "ana@example.com" {
			t.Errorf("use case called with %q %q %q",
				createSurveyor.gotNombre, createSurveyor.gotApellido, createSurveyor.gotEmail)
		}
		if len(createSurveyor.gotRoles) != 1 || createSurveyor.gotRoles[0] != domain.RolAdministrador {
			t.Errorf("actor roles passed to use case = %v", createSurveyor.gotRoles)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		createSurveyor := &createSurveyorStub{}
		recorder := performCreateSurveyorRequest(t, createSurveyor, userVerifierStub{}, "",
			map[string]string{"nombre": "Ana", "apellido": "Gómez", "email": "ana@example.com"})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if createSurveyor.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		createSurveyor := &createSurveyorStub{}
		recorder := performCreateSurveyorRequest(t, createSurveyor, administradorVerifier(), "valid-token", "not-json")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if createSurveyor.called {
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
			{name: "invalid email", err: domain.ErrEmailInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_email"},
			{name: "email in use", err: domain.ErrEmailEnUso, wantStatus: http.StatusConflict, wantCode: "email_in_use"},
			{name: "database unavailable", err: domain.ErrDatabaseUnavailable, wantStatus: http.StatusServiceUnavailable, wantCode: "database_unavailable"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				recorder := performCreateSurveyorRequest(t, &createSurveyorStub{err: test.err}, administradorVerifier(), "valid-token",
					map[string]string{"nombre": "Ana", "apellido": "Gómez", "email": "ana@example.com"})

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
