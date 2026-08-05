package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"loteosapp/backend/internal/infrastructure/auth/keycloak"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type verifierStub struct {
	principal keycloak.Principal
	err       error
}

func (stub verifierStub) Verify(context.Context, string) (keycloak.Principal, error) {
	return stub.principal, stub.err
}

func TestRequireAuth(t *testing.T) {
	t.Parallel()

	wantPrincipal := keycloak.Principal{Subject: "user-123", Roles: []string{"administrador"}}

	tests := []struct {
		name           string
		authorization  string
		verifier       verifierStub
		wantStatus     int
		wantNextCalled bool
	}{
		{
			name:       "missing header",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:          "header without bearer prefix",
			authorization: "token-without-prefix",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "empty bearer token",
			authorization: "Bearer ",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "verifier rejects token",
			authorization: "Bearer invalid",
			verifier:      verifierStub{err: errors.New("signature is invalid")},
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:           "valid token calls next with principal",
			authorization:  "Bearer valid",
			verifier:       verifierStub{principal: wantPrincipal},
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			nextCalled := false
			var gotPrincipal keycloak.Principal
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				gotPrincipal, _ = middleware.PrincipalFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware.RequireAuth(test.verifier)(next)

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.authorization != "" {
				request.Header.Set("Authorization", test.authorization)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if nextCalled != test.wantNextCalled {
				t.Errorf("next called = %v, want %v", nextCalled, test.wantNextCalled)
			}
			if test.wantNextCalled && !reflect.DeepEqual(gotPrincipal, wantPrincipal) {
				t.Errorf("principal = %#v, want %#v", gotPrincipal, wantPrincipal)
			}

			if recorder.Code == http.StatusUnauthorized {
				var got response.ErrorResponse
				if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if got.Code != "unauthorized" {
					t.Errorf("error code = %q, want %q", got.Code, "unauthorized")
				}
			}
		})
	}
}

func TestPrincipalFromContextMissing(t *testing.T) {
	t.Parallel()

	_, ok := middleware.PrincipalFromContext(context.Background())
	if ok {
		t.Error("PrincipalFromContext() ok = true, want false for empty context")
	}
}
