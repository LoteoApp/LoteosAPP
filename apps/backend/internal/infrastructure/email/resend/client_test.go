package resend_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/infrastructure/email/resend"
)

const testAPIKey = "re_test_key"

type fakeResendServer struct {
	mux *http.ServeMux

	status        int
	body          string
	calls         int
	sawAuthHeader bool
	receivedTo    []string
	receivedFrom  string
	receivedHTML  string
	receivedSubj  string
}

func newFakeResendServer(t *testing.T) *fakeResendServer {
	t.Helper()

	fake := &fakeResendServer{mux: http.NewServeMux(), status: http.StatusOK}

	fake.mux.HandleFunc("POST /emails", func(w http.ResponseWriter, r *http.Request) {
		fake.calls++
		fake.sawAuthHeader = r.Header.Get("Authorization") == "Bearer "+testAPIKey

		var payload struct {
			From    string   `json:"from"`
			To      []string `json:"to"`
			Subject string   `json:"subject"`
			HTML    string   `json:"html"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode send email payload: %v", err)
		}
		fake.receivedFrom = payload.From
		fake.receivedTo = payload.To
		fake.receivedSubj = payload.Subject
		fake.receivedHTML = payload.HTML

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(fake.status)
		if fake.body != "" {
			_, _ = w.Write([]byte(fake.body))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "email-1"})
	})

	return fake
}

func testInvite() gateway.UserInviteEmail {
	return gateway.UserInviteEmail{
		To:        "ana@example.com",
		Nombre:    "Ana",
		Apellido:  "Gómez",
		Rol:       domain.RolAdministrativo,
		InviteURL: "https://app.loteosapp.com/aceptar-invitacion?token_hash=abc123&type=invite",
	}
}

func TestClientSendUserInviteHappyPath(t *testing.T) {
	t.Parallel()

	fake := newFakeResendServer(t)
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client, err := resend.NewClient(resend.Config{
		APIKey:    testAPIKey,
		FromEmail: "no-reply@loteosapp.com",
		FromName:  "LoteosAPP",
		BaseURL:   server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := client.SendUserInvite(context.Background(), testInvite()); err != nil {
		t.Fatalf("SendUserInvite() error = %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("SendUserInvite() calls = %d, want 1", fake.calls)
	}
	if !fake.sawAuthHeader {
		t.Error("SendUserInvite() never sent the Authorization header")
	}
	if fake.receivedFrom != "LoteosAPP <no-reply@loteosapp.com>" {
		t.Errorf("SendUserInvite() from = %q", fake.receivedFrom)
	}
	if len(fake.receivedTo) != 1 || fake.receivedTo[0] != "ana@example.com" {
		t.Errorf("SendUserInvite() to = %v", fake.receivedTo)
	}
	if fake.receivedSubj == "" {
		t.Error("SendUserInvite() sent an empty subject")
	}
	for _, want := range []string{"Ana", "Gómez", "administrativo", "https://app.loteosapp.com/aceptar-invitacion?token_hash=abc123&amp;type=invite"} {
		if !strings.Contains(fake.receivedHTML, want) {
			t.Errorf("SendUserInvite() html missing %q, got %q", want, fake.receivedHTML)
		}
	}
}

func TestClientSendUserInviteSurfacesUnexpectedStatus(t *testing.T) {
	t.Parallel()

	fake := newFakeResendServer(t)
	fake.status = http.StatusUnprocessableEntity
	fake.body = `{"statusCode":422,"message":"invalid from address","name":"validation_error"}`
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client, err := resend.NewClient(resend.Config{APIKey: testAPIKey, FromEmail: "no-reply@loteosapp.com", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := client.SendUserInvite(context.Background(), testInvite()); err == nil {
		t.Fatal("SendUserInvite() error = nil, want error on unexpected status")
	}
}

func TestClientSendUserInvitePropagatesTransportError(t *testing.T) {
	t.Parallel()

	fake := newFakeResendServer(t)
	server := httptest.NewServer(fake.mux)
	server.Close()

	client, err := resend.NewClient(resend.Config{APIKey: testAPIKey, FromEmail: "no-reply@loteosapp.com", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := client.SendUserInvite(context.Background(), testInvite()); err == nil {
		t.Error("SendUserInvite() error = nil, want error when the server is unreachable")
	}
}

func testPasswordReset() gateway.PasswordResetEmail {
	return gateway.PasswordResetEmail{
		To:       "ana@example.com",
		Nombre:   "Ana",
		Apellido: "Gómez",
		ResetURL: "https://app.loteosapp.com/restablecer-contrasena?token=abc123",
	}
}

func TestClientSendPasswordResetHappyPath(t *testing.T) {
	t.Parallel()

	fake := newFakeResendServer(t)
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client, err := resend.NewClient(resend.Config{
		APIKey:    testAPIKey,
		FromEmail: "no-reply@loteosapp.com",
		FromName:  "LoteosAPP",
		BaseURL:   server.URL,
	})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := client.SendPasswordReset(context.Background(), testPasswordReset()); err != nil {
		t.Fatalf("SendPasswordReset() error = %v", err)
	}
	if fake.calls != 1 {
		t.Errorf("SendPasswordReset() calls = %d, want 1", fake.calls)
	}
	if len(fake.receivedTo) != 1 || fake.receivedTo[0] != "ana@example.com" {
		t.Errorf("SendPasswordReset() to = %v", fake.receivedTo)
	}
	for _, want := range []string{"Ana", "Gómez", "https://app.loteosapp.com/restablecer-contrasena?token=abc123"} {
		if !strings.Contains(fake.receivedHTML, want) {
			t.Errorf("SendPasswordReset() html missing %q, got %q", want, fake.receivedHTML)
		}
	}
	if strings.Contains(fake.receivedHTML, "Contraseña temporal") {
		t.Error("SendPasswordReset() html should not mention a temporary password")
	}
}

func TestClientSendPasswordResetSurfacesUnexpectedStatus(t *testing.T) {
	t.Parallel()

	fake := newFakeResendServer(t)
	fake.status = http.StatusUnprocessableEntity
	fake.body = `{"statusCode":422,"message":"invalid from address","name":"validation_error"}`
	server := httptest.NewServer(fake.mux)
	t.Cleanup(server.Close)

	client, err := resend.NewClient(resend.Config{APIKey: testAPIKey, FromEmail: "no-reply@loteosapp.com", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := client.SendPasswordReset(context.Background(), testPasswordReset()); err == nil {
		t.Fatal("SendPasswordReset() error = nil, want error on unexpected status")
	}
}

func TestClientSendPasswordResetPropagatesTransportError(t *testing.T) {
	t.Parallel()

	fake := newFakeResendServer(t)
	server := httptest.NewServer(fake.mux)
	server.Close()

	client, err := resend.NewClient(resend.Config{APIKey: testAPIKey, FromEmail: "no-reply@loteosapp.com", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if err := client.SendPasswordReset(context.Background(), testPasswordReset()); err == nil {
		t.Error("SendPasswordReset() error = nil, want error when the server is unreachable")
	}
}

func TestNewClientRequiresApiKeyAndFromEmail(t *testing.T) {
	t.Parallel()

	if _, err := resend.NewClient(resend.Config{}); err == nil {
		t.Error("NewClient() error = nil, want error for an empty config")
	}
	if _, err := resend.NewClient(resend.Config{APIKey: testAPIKey}); err == nil {
		t.Error("NewClient() error = nil, want error when FromEmail is missing")
	}
}
