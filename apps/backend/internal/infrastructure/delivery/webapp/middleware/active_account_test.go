package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type accountRepositoryStub struct {
	usuario domain.Usuario
	err     error
}

func (stub accountRepositoryStub) FindByAuthProviderID(context.Context, string) (domain.Usuario, error) {
	return stub.usuario, stub.err
}

func TestRequireActiveAccountWithNoPrincipal(t *testing.T) {
	t.Parallel()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequireActiveAccount(accountRepositoryStub{})(next)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if nextCalled {
		t.Error("next should not be called without a principal in context")
	}
}

// TestRequireActiveAccount runs the middleware behind RequireAuth, the same
// order route.go wires them in, since PrincipalFromContext only reads what
// RequireAuth's own context key stores.
func TestRequireActiveAccount(t *testing.T) {
	t.Parallel()

	baja := time.Now()

	tests := []struct {
		name           string
		principal      supabase.Principal
		repository     accountRepositoryStub
		wantStatus     int
		wantCode       string
		wantNextCalled bool
	}{
		{
			name:           "administrador claimed by the token with no usuarios row passes through",
			principal:      supabase.Principal{Subject: "auth-1", Roles: []string{domain.RolAdministrador}},
			repository:     accountRepositoryStub{err: domain.ErrUsuarioNoEncontrado},
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:       "non-administrador with no usuarios row is blocked",
			principal:  supabase.Principal{Subject: "auth-1", Roles: []string{"escribano"}},
			repository: accountRepositoryStub{err: domain.ErrUsuarioNoEncontrado},
			wantStatus: http.StatusForbidden,
			wantCode:   "actor_not_provisioned",
		},
		{
			name:       "no usuarios row and no role claim at all is blocked",
			principal:  supabase.Principal{Subject: "auth-1"},
			repository: accountRepositoryStub{err: domain.ErrUsuarioNoEncontrado},
			wantStatus: http.StatusForbidden,
			wantCode:   "actor_not_provisioned",
		},
		{
			name:           "active account passes through",
			principal:      supabase.Principal{Subject: "auth-1"},
			repository:     accountRepositoryStub{usuario: domain.Usuario{ID: "user-1"}},
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:       "inactive account is blocked",
			principal:  supabase.Principal{Subject: "auth-1"},
			repository: accountRepositoryStub{usuario: domain.Usuario{ID: "user-1", FechaBaja: &baja}},
			wantStatus: http.StatusForbidden,
			wantCode:   "account_inactive",
		},
		{
			name:       "unexpected repository error",
			principal:  supabase.Principal{Subject: "auth-1"},
			repository: accountRepositoryStub{err: errors.New("connection refused")},
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "database_unavailable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			nextCalled := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := middleware.RequireAuth(verifierStub{principal: test.principal})(
				middleware.RequireActiveAccount(test.repository)(next))

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer valid")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if nextCalled != test.wantNextCalled {
				t.Errorf("next called = %v, want %v", nextCalled, test.wantNextCalled)
			}
			if test.wantCode != "" {
				var got response.ErrorResponse
				if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if got.Code != test.wantCode {
					t.Errorf("error code = %q, want %q", got.Code, test.wantCode)
				}
			}
		})
	}
}
