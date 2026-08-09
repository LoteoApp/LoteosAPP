package keycloak_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/auth/keycloak"
)

const (
	testRealm        = "loteosapp"
	testClientID     = "loteosapp-backend"
	testClientSecret = "dev-secret"
	testUserID       = "generated-user-id"
)

type fakeAdminServer struct {
	mux *http.ServeMux

	tokenStatus          int
	createUserStatus     int
	resetPasswordStatus  int
	roleLookupStatus     int
	assignRoleStatus     int
	deleteStatus         int
	deleteCalls          int
	sawAuthorizationOnce bool
}

func newFakeAdminServer(t *testing.T) *fakeAdminServer {
	t.Helper()

	fake := &fakeAdminServer{
		mux:                 http.NewServeMux(),
		tokenStatus:         http.StatusOK,
		createUserStatus:    http.StatusCreated,
		resetPasswordStatus: http.StatusNoContent,
		roleLookupStatus:    http.StatusOK,
		assignRoleStatus:    http.StatusNoContent,
		deleteStatus:        http.StatusNoContent,
	}

	fake.mux.HandleFunc("POST /realms/"+testRealm+"/protocol/openid-connect/token", func(w http.ResponseWriter, r *http.Request) {
		if fake.tokenStatus != http.StatusOK {
			w.WriteHeader(fake.tokenStatus)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse token form: %v", err)
		}
		if r.PostForm.Get("grant_type") != "client_credentials" {
			t.Errorf("token grant_type = %q, want client_credentials", r.PostForm.Get("grant_type"))
		}
		if r.PostForm.Get("client_id") != testClientID {
			t.Errorf("token client_id = %q, want %q", r.PostForm.Get("client_id"), testClientID)
		}
		if r.PostForm.Get("client_secret") != testClientSecret {
			t.Errorf("token client_secret = %q, want %q", r.PostForm.Get("client_secret"), testClientSecret)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "admin-token",
			"expires_in":   300,
		})
	})

	fake.mux.HandleFunc("POST /admin/realms/"+testRealm+"/users", func(w http.ResponseWriter, r *http.Request) {
		fake.requireBearer(t, r)

		if fake.createUserStatus == http.StatusCreated {
			w.Header().Set("Location", r.URL.String()+"/"+testUserID)
		}
		w.WriteHeader(fake.createUserStatus)
	})

	fake.mux.HandleFunc("PUT /admin/realms/"+testRealm+"/users/"+testUserID+"/reset-password", func(w http.ResponseWriter, r *http.Request) {
		fake.requireBearer(t, r)
		w.WriteHeader(fake.resetPasswordStatus)
	})

	fake.mux.HandleFunc("GET /admin/realms/"+testRealm+"/roles/{rol}", func(w http.ResponseWriter, r *http.Request) {
		fake.requireBearer(t, r)

		if fake.roleLookupStatus != http.StatusOK {
			w.WriteHeader(fake.roleLookupStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "role-id", "name": r.PathValue("rol")})
	})

	fake.mux.HandleFunc("POST /admin/realms/"+testRealm+"/users/"+testUserID+"/role-mappings/realm", func(w http.ResponseWriter, r *http.Request) {
		fake.requireBearer(t, r)
		w.WriteHeader(fake.assignRoleStatus)
	})

	fake.mux.HandleFunc("DELETE /admin/realms/"+testRealm+"/users/"+testUserID, func(w http.ResponseWriter, r *http.Request) {
		fake.requireBearer(t, r)
		fake.deleteCalls++
		w.WriteHeader(fake.deleteStatus)
	})

	return fake
}

func (fake *fakeAdminServer) requireBearer(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get("Authorization") != "Bearer admin-token" {
		t.Errorf("Authorization = %q, want %q", r.Header.Get("Authorization"), "Bearer admin-token")
	}
	fake.sawAuthorizationOnce = true
}

func TestAdminClientCreateUserHappyPath(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := keycloak.NewAdminClient(server.URL, testRealm, testClientID, testClientSecret)

	keycloakID, temporaryPassword, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if keycloakID != testUserID {
		t.Errorf("CreateUser() id = %q, want %q", keycloakID, testUserID)
	}
	if len(temporaryPassword) < 16 {
		t.Errorf("CreateUser() temporary password too short: %q", temporaryPassword)
	}
	if !fake.sawAuthorizationOnce {
		t.Error("CreateUser() never sent a bearer token")
	}
	if fake.deleteCalls != 0 {
		t.Errorf("CreateUser() deleteCalls = %d, want 0 on success", fake.deleteCalls)
	}
}

func TestAdminClientCreateUserDuplicateEmail(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.createUserStatus = http.StatusConflict
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := keycloak.NewAdminClient(server.URL, testRealm, testClientID, testClientSecret)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)

	if !errors.Is(err, domain.ErrEmailEnUso) {
		t.Fatalf("CreateUser() error = %v, want %v", err, domain.ErrEmailEnUso)
	}
	if fake.deleteCalls != 0 {
		t.Errorf("CreateUser() deleteCalls = %d, want 0 when the account was never created", fake.deleteCalls)
	}
}

func TestAdminClientCreateUserCleansUpWhenRoleAssignmentFails(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.roleLookupStatus = http.StatusNotFound
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := keycloak.NewAdminClient(server.URL, testRealm, testClientID, testClientSecret)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", "rol-inexistente")

	if err == nil {
		t.Fatal("CreateUser() error = nil, want error when the realm role does not exist")
	}
	if fake.deleteCalls != 1 {
		t.Errorf("CreateUser() deleteCalls = %d, want 1 to clean up the half-created account", fake.deleteCalls)
	}
}

func TestAdminClientDeleteUser(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := keycloak.NewAdminClient(server.URL, testRealm, testClientID, testClientSecret)

	if err := client.DeleteUser(context.Background(), testUserID); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if fake.deleteCalls != 1 {
		t.Errorf("DeleteUser() deleteCalls = %d, want 1", fake.deleteCalls)
	}
}

func TestAdminClientDeleteUserSurfacesUnexpectedStatus(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.deleteStatus = http.StatusInternalServerError
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := keycloak.NewAdminClient(server.URL, testRealm, testClientID, testClientSecret)

	if err := client.DeleteUser(context.Background(), testUserID); err == nil {
		t.Error("DeleteUser() error = nil, want error on unexpected status")
	}
}

func TestAdminClientCreateUserCleansUpWhenPasswordResetFails(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.resetPasswordStatus = http.StatusInternalServerError
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := keycloak.NewAdminClient(server.URL, testRealm, testClientID, testClientSecret)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)

	if err == nil {
		t.Fatal("CreateUser() error = nil, want error when setting the temporary password fails")
	}
	if fake.deleteCalls != 1 {
		t.Errorf("CreateUser() deleteCalls = %d, want 1 to clean up the half-created account", fake.deleteCalls)
	}
}

func TestAdminClientCreateUserCleansUpWhenRoleMappingFails(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.assignRoleStatus = http.StatusInternalServerError
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := keycloak.NewAdminClient(server.URL, testRealm, testClientID, testClientSecret)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)

	if err == nil {
		t.Fatal("CreateUser() error = nil, want error when assigning the realm role fails")
	}
	if fake.deleteCalls != 1 {
		t.Errorf("CreateUser() deleteCalls = %d, want 1 to clean up the half-created account", fake.deleteCalls)
	}
}

func TestAdminClientCreateUserPropagatesTokenFailure(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.tokenStatus = http.StatusUnauthorized
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := keycloak.NewAdminClient(server.URL, testRealm, testClientID, testClientSecret)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)

	if err == nil {
		t.Fatal("CreateUser() error = nil, want error when the token request fails")
	}
}
