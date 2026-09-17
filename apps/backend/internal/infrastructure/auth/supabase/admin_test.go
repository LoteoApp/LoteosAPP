package supabase_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
)

const (
	testServiceRoleKey    = "test-service-role-key"
	testUserID            = "11111111-1111-1111-1111-111111111111"
	testHashedToken       = "hashed-token-abc123"
	testInviteRedirectURL = "https://app.loteosapp.com/aceptar-invitacion"
)

type fakeAdminServer struct {
	mux *http.ServeMux

	generateLinkStatus int
	generateLinkBody   string
	generateLinkCalls  int
	lastGenerateLink   struct {
		Type  string `json:"type"`
		Email string `json:"email"`
	}

	putStatus   int
	putBody     string
	putCalls    int
	lastPutBody map[string]any

	deleteStatus int
	deleteCalls  int

	sawAuthHeaders bool
}

func newFakeAdminServer(t *testing.T) *fakeAdminServer {
	t.Helper()

	fake := &fakeAdminServer{
		mux:                http.NewServeMux(),
		generateLinkStatus: http.StatusOK,
		putStatus:          http.StatusOK,
		deleteStatus:       http.StatusNoContent,
	}

	fake.mux.HandleFunc("POST /auth/v1/admin/generate_link", func(w http.ResponseWriter, r *http.Request) {
		fake.requireAuthHeaders(t, r)
		fake.generateLinkCalls++

		if err := json.NewDecoder(r.Body).Decode(&fake.lastGenerateLink); err != nil {
			t.Fatalf("decode generate_link payload: %v", err)
		}

		if fake.generateLinkStatus != http.StatusOK {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(fake.generateLinkStatus)
			if fake.generateLinkBody != "" {
				_, _ = w.Write([]byte(fake.generateLinkBody))
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if fake.generateLinkBody != "" {
			_, _ = w.Write([]byte(fake.generateLinkBody))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": testUserID, "hashed_token": testHashedToken})
	})

	fake.mux.HandleFunc("PUT /auth/v1/admin/users/"+testUserID, func(w http.ResponseWriter, r *http.Request) {
		fake.requireAuthHeaders(t, r)
		fake.putCalls++

		if err := json.NewDecoder(r.Body).Decode(&fake.lastPutBody); err != nil {
			t.Fatalf("decode put payload: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(fake.putStatus)
		if fake.putStatus != http.StatusOK {
			if fake.putBody != "" {
				_, _ = w.Write([]byte(fake.putBody))
			}
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

func newClient(fake *fakeAdminServer, t *testing.T) (*supabase.AdminClient, string) {
	t.Helper()
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)
	return supabase.NewAdminClient(server.URL, testServiceRoleKey, testInviteRedirectURL), server.URL
}

func TestAdminClientCreateUserHappyPath(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	client, _ := newClient(fake, t)

	supabaseID, inviteURL, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if supabaseID != testUserID {
		t.Errorf("CreateUser() id = %q, want %q", supabaseID, testUserID)
	}
	if fake.lastGenerateLink.Type != "invite" || fake.lastGenerateLink.Email != "ana@example.com" {
		t.Errorf("generate_link payload = %+v", fake.lastGenerateLink)
	}

	parsed, err := url.Parse(inviteURL)
	if err != nil {
		t.Fatalf("CreateUser() invite URL is not a valid URL: %v", err)
	}
	if got := parsed.Scheme + "://" + parsed.Host + parsed.Path; got != testInviteRedirectURL {
		t.Errorf("CreateUser() invite URL base = %q, want %q", got, testInviteRedirectURL)
	}
	if parsed.RawQuery != "" {
		t.Errorf("CreateUser() invite URL query = %q, want empty — the token must travel in the fragment", parsed.RawQuery)
	}
	fragment, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		t.Fatalf("CreateUser() invite URL fragment is not valid: %v", err)
	}
	if fragment.Get("token_hash") != testHashedToken {
		t.Errorf("CreateUser() invite URL token_hash = %q, want %q", fragment.Get("token_hash"), testHashedToken)
	}
	if fragment.Get("type") != "invite" {
		t.Errorf("CreateUser() invite URL type = %q, want %q", fragment.Get("type"), "invite")
	}

	if fake.putCalls != 1 {
		t.Fatalf("CreateUser() PUT calls = %d, want 1", fake.putCalls)
	}
	appMetadata, _ := fake.lastPutBody["app_metadata"].(map[string]any)
	if appMetadata["role"] != domain.RolAdministrativo {
		t.Errorf("CreateUser() app_metadata = %+v, want role %q", fake.lastPutBody["app_metadata"], domain.RolAdministrativo)
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
			fake.generateLinkStatus = http.StatusUnprocessableEntity
			fake.generateLinkBody = fmt.Sprintf(`{"code":422,"error_code":%q,"msg":"A user with this email address has already been registered"}`, errorCode)
			client, _ := newClient(fake, t)

			_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)

			if !errors.Is(err, domain.ErrEmailEnUso) {
				t.Fatalf("CreateUser() error = %v, want %v", err, domain.ErrEmailEnUso)
			}
			if fake.putCalls != 0 {
				t.Error("CreateUser() should not set app_metadata when generate_link failed")
			}
		})
	}
}

func TestAdminClientCreateUserSurfacesUnexpectedGenerateLinkStatus(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.generateLinkStatus = http.StatusInternalServerError
	client, _ := newClient(fake, t)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err == nil {
		t.Fatal("CreateUser() error = nil, want error on unexpected status")
	}
}

func TestAdminClientCreateUserPropagatesDecodeError(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.generateLinkBody = "not json"
	client, _ := newClient(fake, t)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err == nil {
		t.Fatal("CreateUser() error = nil, want error when the response body is not valid JSON")
	}
}

func TestAdminClientCreateUserRejectsResponseMissingHashedToken(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.generateLinkBody = fmt.Sprintf(`{"id":%q}`, testUserID)
	client, _ := newClient(fake, t)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err == nil {
		t.Fatal("CreateUser() error = nil, want error when the response is missing a hashed_token")
	}
}

func TestAdminClientCreateUserPropagatesTransportError(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	server := httptest.NewServer(fake.mux)
	server.Close()

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey, testInviteRedirectURL)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err == nil {
		t.Fatal("CreateUser() error = nil, want error when the server is unreachable")
	}
}

// TestAdminClientCreateUserCompensatesWhenAppMetadataFails guards the
// two-call CreateUser: if setting the role after generate_link fails, the
// just-created account must not survive without a role.
func TestAdminClientCreateUserCompensatesWhenAppMetadataFails(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.putStatus = http.StatusInternalServerError
	client, _ := newClient(fake, t)

	_, _, err := client.CreateUser(context.Background(), "ana@example.com", domain.RolAdministrativo)
	if err == nil {
		t.Fatal("CreateUser() error = nil, want error when setting app_metadata fails")
	}
	if fake.deleteCalls != 1 {
		t.Errorf("CreateUser() deleteCalls = %d, want 1 (compensating delete)", fake.deleteCalls)
	}
}

func TestAdminClientDeleteUserPropagatesTransportError(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	server := httptest.NewServer(fake.mux)
	server.Close()

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey, testInviteRedirectURL)

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
			client, _ := newClient(fake, t)

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
	client, _ := newClient(fake, t)

	if err := client.DeleteUser(context.Background(), testUserID); err == nil {
		t.Error("DeleteUser() error = nil, want error on unexpected status")
	}
}

func TestAdminClientGenerateInviteLink(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	client, _ := newClient(fake, t)

	inviteURL, err := client.GenerateInviteLink(context.Background(), "ana@example.com")
	if err != nil {
		t.Fatalf("GenerateInviteLink() error = %v", err)
	}
	if fake.lastGenerateLink.Type != "invite" || fake.lastGenerateLink.Email != "ana@example.com" {
		t.Errorf("generate_link payload = %+v", fake.lastGenerateLink)
	}
	if fake.putCalls != 0 {
		t.Error("GenerateInviteLink() should not touch app_metadata for an existing account")
	}

	parsed, err := url.Parse(inviteURL)
	if err != nil {
		t.Fatalf("GenerateInviteLink() invite URL is not a valid URL: %v", err)
	}
	fragment, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		t.Fatalf("GenerateInviteLink() invite URL fragment is not valid: %v", err)
	}
	if fragment.Get("token_hash") != testHashedToken {
		t.Errorf("GenerateInviteLink() invite URL token_hash = %q, want %q", fragment.Get("token_hash"), testHashedToken)
	}
}

func TestAdminClientGenerateInviteLinkSurfacesUnexpectedStatus(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.generateLinkStatus = http.StatusInternalServerError
	client, _ := newClient(fake, t)

	if _, err := client.GenerateInviteLink(context.Background(), "ana@example.com"); err == nil {
		t.Error("GenerateInviteLink() error = nil, want error on unexpected status")
	}
}

func TestAdminClientGenerateInviteLinkPropagatesTransportError(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	server := httptest.NewServer(fake.mux)
	server.Close()

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey, testInviteRedirectURL)

	if _, err := client.GenerateInviteLink(context.Background(), "ana@example.com"); err == nil {
		t.Error("GenerateInviteLink() error = nil, want error when the server is unreachable")
	}
}

func TestAdminClientSetPassword(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	client, _ := newClient(fake, t)

	if err := client.SetPassword(context.Background(), testUserID, "a-chosen-password"); err != nil {
		t.Fatalf("SetPassword() error = %v", err)
	}
	if fake.putCalls != 1 {
		t.Errorf("SetPassword() calls = %d, want 1", fake.putCalls)
	}
	if !fake.sawAuthHeaders {
		t.Error("SetPassword() never sent apikey/Authorization headers")
	}
}

func TestAdminClientSetPasswordRejectsWeakPassword(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.putStatus = http.StatusUnprocessableEntity
	fake.putBody = `{"code":422,"error_code":"weak_password","msg":"Password should be at least 6 characters"}`
	client, _ := newClient(fake, t)

	err := client.SetPassword(context.Background(), testUserID, "123")

	if !errors.Is(err, domain.ErrPasswordInvalido) {
		t.Fatalf("SetPassword() error = %v, want %v", err, domain.ErrPasswordInvalido)
	}
}

func TestAdminClientSetPasswordSurfacesUnexpectedStatus(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	fake.putStatus = http.StatusInternalServerError
	client, _ := newClient(fake, t)

	if err := client.SetPassword(context.Background(), testUserID, "a-chosen-password"); err == nil {
		t.Error("SetPassword() error = nil, want error on unexpected status")
	}
}

func TestAdminClientSetPasswordPropagatesTransportError(t *testing.T) {
	t.Parallel()

	fake := newFakeAdminServer(t)
	server := httptest.NewServer(fake.mux)
	server.Close()

	client := supabase.NewAdminClient(server.URL, testServiceRoleKey, testInviteRedirectURL)

	if err := client.SetPassword(context.Background(), testUserID, "a-chosen-password"); err == nil {
		t.Error("SetPassword() error = nil, want error when the server is unreachable")
	}
}
