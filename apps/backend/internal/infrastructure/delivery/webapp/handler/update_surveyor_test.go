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

const surveyorID = "0f8fad5b-d9cb-469f-a165-70867728950e"

type updateSurveyorStub struct {
	usuario     domain.Usuario
	err         error
	called      bool
	gotSubject  string
	gotID       string
	gotNombre   *string
	gotApellido *string
}

func (stub *updateSurveyorStub) Execute(_ context.Context, _ []string, subject, id string, nombre, apellido *string) (domain.Usuario, error) {
	stub.called = true
	stub.gotSubject = subject
	stub.gotID = id
	stub.gotNombre = nombre
	stub.gotApellido = apellido
	return stub.usuario, stub.err
}

func performUpdateSurveyorRequest(t *testing.T, updateSurveyor *updateSurveyorStub, verifier userVerifierStub, token, id string, body any) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/v1/agrimensores/{id}", middleware.RequireAuth(verifier)(
		handler.Adapt(handler.NewUpdateSurveyorHandler(updateSurveyor), 5*time.Second)))

	return performRequest(t, mux, http.MethodPatch, "/api/v1/agrimensores/"+id, token, body)
}

func TestUpdateSurveyorRoute(t *testing.T) {
	t.Parallel()

	t.Run("updates the agrimensor", func(t *testing.T) {
		t.Parallel()

		updateSurveyor := &updateSurveyorStub{usuario: domain.Usuario{
			ID: surveyorID, Nombre: "Ana María", Apellido: "Gómez", Rol: domain.RolAgrimensor,
		}}

		recorder := performUpdateSurveyorRequest(t, updateSurveyor, administradorVerifier(), "valid-token", surveyorID,
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
		if got.ID != surveyorID || got.Nombre != "Ana María" {
			t.Errorf("response = %#v", got)
		}
		if updateSurveyor.gotID != surveyorID {
			t.Errorf("use case called with id %q, want %q", updateSurveyor.gotID, surveyorID)
		}
		if updateSurveyor.gotSubject != "admin-1" {
			t.Errorf("use case called with subject %q, want %q", updateSurveyor.gotSubject, "admin-1")
		}
		if updateSurveyor.gotNombre == nil || *updateSurveyor.gotNombre != "Ana María" {
			t.Errorf("nombre passed to use case = %v", updateSurveyor.gotNombre)
		}
		if updateSurveyor.gotApellido != nil {
			t.Error("an omitted field should reach the use case as nil")
		}
	})

	t.Run("rejects an id that is not a uuid", func(t *testing.T) {
		t.Parallel()

		updateSurveyor := &updateSurveyorStub{}
		recorder := performUpdateSurveyorRequest(t, updateSurveyor, administradorVerifier(), "valid-token", "not-a-uuid",
			map[string]string{"nombre": "Ana"})

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}

		var got response.ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Code != "invalid_surveyor_id" {
			t.Errorf("error code = %q, want %q", got.Code, "invalid_surveyor_id")
		}
		if updateSurveyor.called {
			t.Error("use case should not be called with a malformed id")
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		updateSurveyor := &updateSurveyorStub{}
		recorder := performUpdateSurveyorRequest(t, updateSurveyor, userVerifierStub{}, "", surveyorID,
			map[string]string{"nombre": "Ana"})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if updateSurveyor.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		updateSurveyor := &updateSurveyorStub{}
		recorder := performUpdateSurveyorRequest(t, updateSurveyor, administradorVerifier(), "valid-token", surveyorID, "not-json")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if updateSurveyor.called {
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
			{name: "not found", err: domain.ErrAgrimensorNoEncontrado, wantStatus: http.StatusNotFound, wantCode: "surveyor_not_found"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				recorder := performUpdateSurveyorRequest(t, &updateSurveyorStub{err: test.err}, administradorVerifier(),
					"valid-token", surveyorID, map[string]string{"nombre": "Ana"})

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
