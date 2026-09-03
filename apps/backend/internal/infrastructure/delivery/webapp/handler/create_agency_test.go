package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/agencies"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type createAgencyStub struct {
	inmobiliaria   domain.Inmobiliaria
	err            error
	called         bool
	gotActorRoles  []string
	gotSubject     string
	gotRazonSocial string
	gotCUIT        *string
	gotTelefono    *string
	gotEmail       *string
}

func (stub *createAgencyStub) Execute(_ context.Context, input agencies.CreateAgencyInput) (domain.Inmobiliaria, error) {
	stub.called = true
	stub.gotActorRoles = input.ActorRoles
	stub.gotSubject = input.Subject
	stub.gotRazonSocial = input.RazonSocial
	stub.gotCUIT = input.CUIT
	stub.gotTelefono = input.Telefono
	stub.gotEmail = input.Email
	return stub.inmobiliaria, stub.err
}

func performCreateAgencyRequest(t *testing.T, createAgency *createAgencyStub, verifier userVerifierStub, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewCreateAgencyHandler(createAgency)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/inmobiliarias", requireAuth(handler.Adapt(h, 5*time.Second)))

	return performRequest(t, mux, http.MethodPost, "/api/v1/inmobiliarias", token, body)
}

func TestCreateAgencyRoute(t *testing.T) {
	t.Parallel()

	t.Run("creates the agency when the actor is administrador", func(t *testing.T) {
		t.Parallel()

		createAgency := &createAgencyStub{
			inmobiliaria: domain.Inmobiliaria{ID: validAgencyID, RazonSocial: "Lotes del Sur"},
		}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performCreateAgencyRequest(t, createAgency, verifier, "valid-token",
			map[string]string{"razonSocial": "Lotes del Sur", "cuit": "30-71234567-8"})

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
		}

		var got struct {
			ID          string `json:"id"`
			RazonSocial string `json:"razonSocial"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != validAgencyID || got.RazonSocial != "Lotes del Sur" {
			t.Errorf("response = %#v", got)
		}
		if createAgency.gotSubject != "user-1" {
			t.Errorf("subject passed to use case = %q, want %q", createAgency.gotSubject, "user-1")
		}
		if len(createAgency.gotActorRoles) != 1 || createAgency.gotActorRoles[0] != domain.RolAdministrador {
			t.Errorf("actor roles passed to use case = %v", createAgency.gotActorRoles)
		}
		if createAgency.gotCUIT == nil || *createAgency.gotCUIT != "30-71234567-8" {
			t.Errorf("cuit passed to use case = %v, want it forwarded verbatim", createAgency.gotCUIT)
		}
	})

	t.Run("forwards omitted optional fields as absent", func(t *testing.T) {
		t.Parallel()

		createAgency := &createAgencyStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performCreateAgencyRequest(t, createAgency, verifier, "valid-token",
			map[string]string{"razonSocial": "Lotes del Sur"})

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
		}
		if createAgency.gotCUIT != nil || createAgency.gotTelefono != nil || createAgency.gotEmail != nil {
			t.Errorf("optional fields passed to use case = %v/%v/%v, want all absent",
				createAgency.gotCUIT, createAgency.gotTelefono, createAgency.gotEmail)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		createAgency := &createAgencyStub{}
		recorder := performCreateAgencyRequest(t, createAgency, userVerifierStub{}, "",
			map[string]string{"razonSocial": "Lotes del Sur"})

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if createAgency.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		createAgency := &createAgencyStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performCreateAgencyRequest(t, createAgency, verifier, "valid-token", "not-json")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if createAgency.called {
			t.Error("use case should not be called with an invalid body")
		}
	})

	// The body limit runs before the use case, so an authenticated caller
	// with no permission still can't make the decoder allocate.
	t.Run("rejects an oversized body without reaching the use case", func(t *testing.T) {
		t.Parallel()

		createAgency := &createAgencyStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolEscribano}}}

		oversized := `{"razonSocial":"` + strings.Repeat("a", 64<<10) + `"}`
		recorder := performCreateAgencyRequest(t, createAgency, verifier, "valid-token", oversized)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
		if createAgency.called {
			t.Error("use case should not be called with an oversized body")
		}

		var got response.ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Code != "invalid_body" {
			t.Errorf("error code = %q, want %q", got.Code, "invalid_body")
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
			{name: "invalid agency", err: domain.ErrInmobiliariaInvalida, wantStatus: http.StatusBadRequest, wantCode: "invalid_agency"},
			{name: "invalid cuit", err: domain.ErrCUITInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_cuit"},
			{name: "cuit in use", err: domain.ErrCUITEnUso, wantStatus: http.StatusConflict, wantCode: "cuit_in_use"},
			{name: "invalid email", err: domain.ErrEmailInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_email"},
			{name: "actor not provisioned", err: domain.ErrActorNoAprovisionado, wantStatus: http.StatusForbidden, wantCode: "actor_not_provisioned"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				createAgency := &createAgencyStub{err: test.err}
				verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

				recorder := performCreateAgencyRequest(t, createAgency, verifier, "valid-token",
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
