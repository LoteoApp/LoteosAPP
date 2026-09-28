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
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type resetPasswordStub struct {
	err       error
	called    bool
	gotToken  string
	gotPasswd string
}

func (stub *resetPasswordStub) Execute(_ context.Context, token, newPassword string) error {
	stub.called = true
	stub.gotToken = token
	stub.gotPasswd = newPassword
	return stub.err
}

func performResetPasswordRequest(t *testing.T, resetPassword *resetPasswordStub, body any) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/auth/restablecer-contrasena", handler.Adapt(handler.NewResetPasswordHandler(resetPassword), 5*time.Second))

	return performRequest(t, mux, http.MethodPost, "/api/v1/auth/restablecer-contrasena", "", body)
}

func TestResetPasswordRoute(t *testing.T) {
	t.Parallel()

	t.Run("answers 204 with no body", func(t *testing.T) {
		t.Parallel()

		resetPassword := &resetPasswordStub{}
		recorder := performResetPasswordRequest(t, resetPassword, map[string]string{"token": "abc123", "newPassword": "a-new-password"})

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
		}
		if recorder.Body.Len() != 0 {
			t.Errorf("body = %q, want empty", recorder.Body.String())
		}
		if resetPassword.gotToken != "abc123" || resetPassword.gotPasswd != "a-new-password" {
			t.Errorf("use case called with token %q, password %q", resetPassword.gotToken, resetPassword.gotPasswd)
		}
	})

	t.Run("does not require authentication", func(t *testing.T) {
		t.Parallel()

		resetPassword := &resetPasswordStub{}
		recorder := performResetPasswordRequest(t, resetPassword, map[string]string{"token": "abc123", "newPassword": "a-new-password"})

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d — this route must work without a token", recorder.Code, http.StatusNoContent)
		}
		if !resetPassword.called {
			t.Error("use case should be called even without a bearer token")
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
			{name: "invalid password", err: domain.ErrPasswordInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_password"},
			{name: "invalid token", err: domain.ErrPasswordResetTokenInvalido, wantStatus: http.StatusBadRequest, wantCode: "password_reset_token_invalid"},
			{name: "inactive user", err: domain.ErrUsuarioDadoDeBaja, wantStatus: http.StatusConflict, wantCode: "user_already_inactive"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				recorder := performResetPasswordRequest(t, &resetPasswordStub{err: test.err}, map[string]string{"token": "abc123", "newPassword": "a-new-password"})

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
