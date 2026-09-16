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

type deleteAgencyStub struct {
	err        error
	called     bool
	gotSubject string
	gotID      string
}

func (stub *deleteAgencyStub) Execute(_ context.Context, input agencies.DeleteAgencyInput) error {
	stub.called = true
	stub.gotSubject = input.Subject
	stub.gotID = input.ID
	return stub.err
}

func performDeleteAgencyRequest(t *testing.T, deleteAgency *deleteAgencyStub, verifier userVerifierStub, token, id string) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewDeleteAgencyHandler(deleteAgency)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/v1/inmobiliarias/{id}", requireAuth(handler.Adapt(h, 5*time.Second)))

	return performRequest(t, mux, http.MethodDelete, "/api/v1/inmobiliarias/"+id, token, nil)
}

func TestDeleteAgencyRoute(t *testing.T) {
	t.Parallel()

	t.Run("gives the agency de baja", func(t *testing.T) {
		t.Parallel()

		deleteAgency := &deleteAgencyStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performDeleteAgencyRequest(t, deleteAgency, verifier, "valid-token", validAgencyID)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
		}
		if deleteAgency.gotID != validAgencyID {
			t.Errorf("id passed to use case = %q, want %q", deleteAgency.gotID, validAgencyID)
		}
		if deleteAgency.gotSubject != "user-1" {
			t.Errorf("subject passed to use case = %q, want %q", deleteAgency.gotSubject, "user-1")
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		deleteAgency := &deleteAgencyStub{}
		recorder := performDeleteAgencyRequest(t, deleteAgency, userVerifierStub{}, "", validAgencyID)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if deleteAgency.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects a non-uuid id before hitting the use case, without touching PostgreSQL", func(t *testing.T) {
		t.Parallel()

		deleteAgency := &deleteAgencyStub{}
		verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrador}}}

		recorder := performDeleteAgencyRequest(t, deleteAgency, verifier, "valid-token", "not-a-uuid")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
		if deleteAgency.called {
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

	t.Run("maps use case errors to the right status", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "not authorized", err: domain.ErrNoAutorizado, wantStatus: http.StatusForbidden, wantCode: "forbidden"},
			{name: "agency not found", err: domain.ErrAgencyNotFound, wantStatus: http.StatusNotFound, wantCode: "agency_not_found"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				deleteAgency := &deleteAgencyStub{err: test.err}
				verifier := userVerifierStub{principal: supabase.Principal{Subject: "user-1", Roles: []string{domain.RolAdministrativo}}}

				recorder := performDeleteAgencyRequest(t, deleteAgency, verifier, "valid-token", validAgencyID)

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
