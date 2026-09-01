package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/agencies"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

const validAgencyID = "22222222-2222-2222-2222-222222222222"

type updateAgencyStub struct {
	inmobiliaria   domain.Inmobiliaria
	err            error
	called         bool
	gotSubject     string
	gotID          string
	gotRazonSocial *string
	gotCUIT        *string
	gotTelefono    *string
	gotEmail       *string
}

func (stub *updateAgencyStub) Execute(_ context.Context, input agencies.UpdateAgencyInput) (domain.Inmobiliaria, error) {
	stub.called = true
	stub.gotSubject = input.Subject
	stub.gotID = input.ID
	stub.gotRazonSocial = input.RazonSocial
	stub.gotCUIT = input.CUIT
	stub.gotTelefono = input.Telefono
	stub.gotEmail = input.Email
	return stub.inmobiliaria, stub.err
}

func performUpdateAgencyRequest(t *testing.T, updateAgency *updateAgencyStub, verifier userVerifierStub, token, id string, body any) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewUpdateAgencyHandler(updateAgency)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("PATCH /api/v1/inmobiliarias/{id}", requireAuth(handler.Adapt(h, 5*time.Second)))

	return performRequest(t, mux, http.MethodPatch, "/api/v1/inmobiliarias/"+id, token, body)
}

func TestUpdateAgencyRoute(t *testing.T) {
	t.Parallel()

	t.Run("updates the agency", func(t *testing.T) {
		t.Parallel()

		updateAgency := &updateAgencyStub{
			inmobiliaria: domain.Inmobiliaria{ID: validAgencyID, RazonSocial: "Lotes del Sur SRL"},
		}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performUpdateAgencyRequest(t, updateAgency, verifier, "valid-token", validAgencyID,
			map[string]string{"razonSocial": "Lotes del Sur SRL"})

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if updateAgency.gotID != validAgencyID {
			t.Errorf("id passed to use case = %q, want %q", updateAgency.gotID, validAgencyID)
		}
		if updateAgency.gotSubject != "user-1" {
			t.Errorf("subject passed to use case = %q, want %q", updateAgency.gotSubject, "user-1")
		}
		if updateAgency.gotRazonSocial == nil || *updateAgency.gotRazonSocial != "Lotes del Sur SRL" {
			t.Errorf("razon social passed to use case = %v", updateAgency.gotRazonSocial)
		}

		var got struct {
			RazonSocial string `json:"razonSocial"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.RazonSocial != "Lotes del Sur SRL" {
			t.Errorf("response = %#v", got)
		}
	})

	// A PATCH must not turn the fields the caller left out into a change,
	// so what the handler forwards for them has to stay absent.
	t.Run("forwards omitted fields as absent", func(t *testing.T) {
		t.Parallel()

		updateAgency := &updateAgencyStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performUpdateAgencyRequest(t, updateAgency, verifier, "valid-token", validAgencyID,
			map[string]string{"telefono": "3415551234"})

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if updateAgency.gotRazonSocial != nil || updateAgency.gotCUIT != nil || updateAgency.gotEmail != nil {
			t.Errorf("omitted fields passed to use case = %v/%v/%v, want all absent",
				updateAgency.gotRazonSocial, updateAgency.gotCUIT, updateAgency.gotEmail)
		}
		if updateAgency.gotTelefono == nil || *updateAgency.gotTelefono != "3415551234" {
			t.Errorf("telefono passed to use case = %v", updateAgency.gotTelefono)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		updateAgency := &updateAgencyStub{}
		recorder := performUpdateAgencyRequest(t, updateAgency, userVerifierStub{}, "", validAgencyID,
			map[string]string{"razonSocial": "Lotes del Sur"})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if updateAgency.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects a non-uuid id before hitting the use case, without touching PostgreSQL", func(t *testing.T) {
		t.Parallel()

		updateAgency := &updateAgencyStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performUpdateAgencyRequest(t, updateAgency, verifier, "valid-token", "not-a-uuid",
			map[string]string{"razonSocial": "Lotes del Sur"})

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
		if updateAgency.called {
			t.Error("use case should not be called with a non-uuid id")
		}

		var got response.ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Code != "invalid_agency_id" {
			t.Errorf("error code = %q, want %q", got.Code, "invalid_agency_id")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		updateAgency := &updateAgencyStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performUpdateAgencyRequest(t, updateAgency, verifier, "valid-token", validAgencyID, "not-json")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if updateAgency.called {
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
			{name: "agency not found", err: domain.ErrInmobiliariaNoEncontrada, wantStatus: http.StatusNotFound, wantCode: "agency_not_found"},
			{name: "empty update", err: domain.ErrInmobiliariaSinCambios, wantStatus: http.StatusBadRequest, wantCode: "empty_agency_update"},
			{name: "cuit in use", err: domain.ErrCUITEnUso, wantStatus: http.StatusConflict, wantCode: "cuit_in_use"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				updateAgency := &updateAgencyStub{err: test.err}
				verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

				recorder := performUpdateAgencyRequest(t, updateAgency, verifier, "valid-token", validAgencyID,
					map[string]string{"razonSocial": "Lotes del Sur"})

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
