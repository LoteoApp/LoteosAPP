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

type deactivateUserStub struct {
	err        error
	called     bool
	gotSubject string
	gotID      string
}

func (stub *deactivateUserStub) Execute(_ context.Context, _ []string, subject, id string) error {
	stub.called = true
	stub.gotSubject = subject
	stub.gotID = id
	return stub.err
}

func performDeactivateUserRequest(t *testing.T, deactivateUser *deactivateUserStub, verifier userVerifierStub, token, id string) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/v1/usuarios/{id}", middleware.RequireAuth(verifier)(
		handler.Adapt(handler.NewDeactivateUserHandler(deactivateUser), 5*time.Second)))

	return performRequest(t, mux, http.MethodDelete, "/api/v1/usuarios/"+id, token, nil)
}

func TestDeactivateUserRoute(t *testing.T) {
	t.Parallel()

	t.Run("answers 204 with no body", func(t *testing.T) {
		t.Parallel()

		deactivateUser := &deactivateUserStub{}
		recorder := performDeactivateUserRequest(t, deactivateUser, administradorVerifier(), "valid-token", managedUserID)

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
		}
		if recorder.Body.Len() != 0 {
			t.Errorf("body = %q, want empty", recorder.Body.String())
		}
		if deactivateUser.gotID != managedUserID || deactivateUser.gotSubject != "admin-1" {
			t.Errorf("use case called with id %q subject %q", deactivateUser.gotID, deactivateUser.gotSubject)
		}
	})

	t.Run("rejects an id that is not a uuid", func(t *testing.T) {
		t.Parallel()

		deactivateUser := &deactivateUserStub{}
		recorder := performDeactivateUserRequest(t, deactivateUser, administradorVerifier(), "valid-token", "not-a-uuid")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if deactivateUser.called {
			t.Error("use case should not be called with a malformed id")
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		deactivateUser := &deactivateUserStub{}
		recorder := performDeactivateUserRequest(t, deactivateUser, userVerifierStub{}, "", managedUserID)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if deactivateUser.called {
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
			{name: "already inactive", err: domain.ErrUsuarioDadoDeBaja, wantStatus: http.StatusConflict, wantCode: "user_already_inactive"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				recorder := performDeactivateUserRequest(t, &deactivateUserStub{err: test.err},
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
