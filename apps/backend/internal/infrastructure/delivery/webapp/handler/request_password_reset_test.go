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

type requestPasswordResetStub struct {
	err     error
	called  bool
	gotMail string
}

func (stub *requestPasswordResetStub) Execute(_ context.Context, email string) error {
	stub.called = true
	stub.gotMail = email
	return stub.err
}

func performRequestPasswordResetRequest(t *testing.T, requestPasswordReset *requestPasswordResetStub, body any) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/auth/recuperar-contrasena", handler.Adapt(handler.NewRequestPasswordResetHandler(requestPasswordReset), 5*time.Second))

	return performRequest(t, mux, http.MethodPost, "/api/v1/auth/recuperar-contrasena", "", body)
}

func TestRequestPasswordResetRoute(t *testing.T) {
	t.Parallel()

	t.Run("answers 204 with no body", func(t *testing.T) {
		t.Parallel()

		requestPasswordReset := &requestPasswordResetStub{}
		recorder := performRequestPasswordResetRequest(t, requestPasswordReset, map[string]string{"email": "ana@example.com"})

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusNoContent, recorder.Body.String())
		}
		if recorder.Body.Len() != 0 {
			t.Errorf("body = %q, want empty", recorder.Body.String())
		}
		if requestPasswordReset.gotMail != "ana@example.com" {
			t.Errorf("use case called with email %q, want %q", requestPasswordReset.gotMail, "ana@example.com")
		}
	})

	t.Run("does not require authentication", func(t *testing.T) {
		t.Parallel()

		requestPasswordReset := &requestPasswordResetStub{}
		recorder := performRequestPasswordResetRequest(t, requestPasswordReset, map[string]string{"email": "ana@example.com"})

		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d — this route must work without a token", recorder.Code, http.StatusNoContent)
		}
		if !requestPasswordReset.called {
			t.Error("use case should be called even without a token")
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
			{name: "invalid email", err: domain.ErrEmailInvalido, wantStatus: http.StatusBadRequest, wantCode: "invalid_email"},
			{name: "rate limited", err: domain.ErrPasswordResetRateLimited, wantStatus: http.StatusTooManyRequests, wantCode: "password_reset_rate_limited"},
			{name: "busy", err: domain.ErrPasswordResetBusy, wantStatus: http.StatusServiceUnavailable, wantCode: "password_reset_busy"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				recorder := performRequestPasswordResetRequest(t, &requestPasswordResetStub{err: test.err}, map[string]string{"email": "ana@example.com"})

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
