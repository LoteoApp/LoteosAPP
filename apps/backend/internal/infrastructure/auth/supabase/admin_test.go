package supabase_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
)

const (
	testServiceRoleKey = "test-service-role-key"
	testUserID         = "11111111-1111-1111-1111-111111111111"
)

type fakeAdminServer struct {
	mux *http.ServeMux

	createUserStatus int
	createUserBody   string
	deleteStatus     int
	deleteCalls      int
	sawAuthHeaders   bool
}

func newFakeAdminServer(t *testing.T) *fakeAdminServer {
	t.Helper()

	fake := &fakeAdminServer{
		mux:              http.NewServeMux(),
		createUserStatus: http.StatusOK,
		deleteStatus:     http.StatusNoContent,
	}

	fake.mux.HandleFunc("POST /auth/v1/admin/users", func(w http.ResponseWriter, r *http.Request) {
		fake.requireAuthHeaders(t, r)

		if fake.createUserStatus != http.StatusOK {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(fake.createUserStatus)
			if fake.createUserBody != "" {
				_, _ = w.Write([]byte(fake.createUserBody))
			}
			return
		}

		var payload struct {
			Email       string            `json:"email"`
			AppMetadata map[string]string `json:"app_metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode create user payload: %v", err)
		}
		if payload.Email == "" {
			t.Error("create user payload missing email")
		}
		if payload.AppMetadata["role"] == "" {
			t.Error("create user payload missing app_metadata.role")
		}

		w.Header().Set("Content-Type", "application/json")
		if fake.createUserBody != "" {
			_, _ = w.Write([]byte(fake.createUserBody))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": testUserID})
	})

	fake.mux.HandleFunc("DELETE /auth/v1/admin/users/"+testUserID, func(w http.ResponseWriter, r *http.Request) {
		fake.requireAuthHeaders(t, r)
		fake.deleteCalls++
		w.WriteHeader(fake.deleteStatus)
	})

	return fake
}

func (fake *fakeAdminServer) requireAuthHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get("apikey") != testServiceRoleKey {
		t.Errorf("apikey = %q, want %q", r.Header.Get("apikey"), testServiceRoleKey)
	}
	if r.Header.Get("Authorization") != "Bearer "+testServiceRoleKey {
		t.Errorf("Authorization = %q, want %q", r.Header.Get("Authorization"), "Bearer "+testServiceRoleKey)
	}
	fake.sawAuthHeaders = true
}

func TestAdminClientCreateUserHappyPath(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey)

	supabaseID, temporaryPassword, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if supabaseID != testUserID {
		t.Errorf("CreateUser() id = %q, want %q", supabaseID, testUserID)
	}
	if len(temporaryPassword) < 16 {
		t.Errorf("CreateUser() temporary password too short: %q", temporaryPassword)
	}
	if !fake.sawAuthHeaders {
		t.Error("CreateUser() never sent apikey/Authorization headers")
	}
}

func TestAdminClientCreateUserDuplicateEmail(t *testing.T) {
	t.Parallel()

	for _, errorCode := range []string{"email_exists", "user_already_exists"} {
		t.Run(errorCode, func(t *testing.T) {
			t.Parallel()

			fake := newFakeAdminServer(t)
			fake.createUserStatus = http.StatusUnprocessableEntity
			fake.createUserBody = fmt.Sprintf(`{"code":422,"error_code":%q,"msg":"A user with this email address has already been registered"}`, errorCode)
			server := httptest.NewServer(fake.mux)
			t.Cleanup(server.Close)

			client := supabase.NewAdminClient(server.URL, testServiceRoleKey)

			_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)

			if !errors.Is(err, domain.ErrEmailEnUso) {
				t.Fatalf("CreateUser() error = %v, want %v", err, domain.ErrEmailEnUso)
			}
		})
	}
}

// TestAdminClientCreateUserWeakPasswordIsNotDuplicateEmail guards against
// mapping every 422 to ErrEmailEnUso: GoTrue also returns 422 for a weak
// password, which must surface as a distinct, generic error instead of
// telling the caller the email is already registered.
func TestAdminClientCreateUserWeakPasswordIsNotDuplicateEmail(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.createUserStatus = http.StatusUnprocessableEntity
	fake.createUserBody = `{"code":422,"error_code":"weak_password","msg":"Password should be at least 6 characters"}`
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)

	if err == nil {
		t.Fatal("CreateUser() error = nil, want error for a weak password")
	}
	if errors.Is(err, domain.ErrEmailEnUso) {
		t.Fatalf("CreateUser() error = %v, want a generic error, not %v", err, domain.ErrEmailEnUso)
	}
}

func TestAdminClientCreateUserSurfacesUnexpectedStatus(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.createUserStatus = http.StatusInternalServerError
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err == nil {
		t.Fatal("CreateUser() error = nil, want error on unexpected status")
	}
}

func TestAdminClientCreateUserPropagatesDecodeError(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.createUserBody = "not json"
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err == nil {
		t.Fatal("CreateUser() error = nil, want error when the response body is not valid JSON")
	}
}

func TestAdminClientCreateUserRejectsResponseMissingID(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.createUserBody = "{}"
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err == nil {
		t.Fatal("CreateUser() error = nil, want error when the response is missing an id")
	}
}

func TestAdminClientCreateUserPropagatesTransportError(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	server := httptest.NewServer(fake.mux)
	server.Close()

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err == nil {
		t.Fatal("CreateUser() error = nil, want error when the server is unreachable")
	}
}

func TestAdminClientDeleteUserPropagatesTransportError(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	server := httptest.NewServer(fake.mux)
	server.Close()

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey)

	if err := client.DeleteUser(context.Background(), testUserID); err == nil {
		t.Error("DeleteUser() error = nil, want error when the server is unreachable")
	}
}

func TestAdminClientDeleteUser(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusOK, http.StatusNoContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			fake := newFakeAdminServer(t)
			fake.deleteStatus = status
			server := httptest.NewServer(fake.mux)
			t.Cleanup(server.Close)

			client := supabase.NewAdminClient(server.URL, testServiceRoleKey)

			if err := client.DeleteUser(context.Background(), testUserID); err != nil {
				t.Fatalf("DeleteUser() error = %v", err)
			}
			if fake.deleteCalls != 1 {
				t.Errorf("DeleteUser() deleteCalls = %d, want 1", fake.deleteCalls)
			}
		})
	}
}

func TestAdminClientDeleteUserSurfacesUnexpectedStatus(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.deleteStatus = http.StatusInternalServerError
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey)

	if err := client.DeleteUser(context.Background(), testUserID); err == nil {
		t.Error("DeleteUser() error = nil, want error on unexpected status")
	}
}
