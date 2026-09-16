package supabase_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"loteosapp/backend/internal/infrastructure/auth/supabase"
)

const testKeyID = "test-key"

func TestVerifierVerify(t *testing.T) {
	t.Parallel()

	key, otherKey := generateRSAKey(t), generateRSAKey(t)
	server := newJWKSServer(t, key)
	t.Cleanup(server.Close)

	issuer := server.URL + "/auth/v1"

	newVerifier := func(t *testing.T) *supabase.Verifier {
		t.Helper()

		verifier, err := supabase.NewVerifier(context.Background(), server.URL)
		if err != nil {
			t.Fatalf("NewVerifier() error = %v", err)
		}
		t.Cleanup(verifier.Close)

		return verifier
	}

	t.Run("valid token extracts subject, email and role from app_metadata", func(t *testing.T) {
		t.Parallel()

		verifier := newVerifier(t)
		token := signToken(t, key, jwt.MapClaims{
			"iss":   issuer,
			"aud":   "authenticated",
			"sub":   "user-123",
			"email": "ana@example.com",
			"exp":   jwt.NewNumericDate(time.Now().Add(time.Hour)),
			"app_metadata": map[string]any{
				"role": "administrador",
			},
		})

		principal, err := verifier.Verify(context.Background(), token)
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}

		if principal.Subject != "user-123" {
			t.Errorf("Subject = %q, want %q", principal.Subject, "user-123")
		}
		if principal.Email != "ana@example.com" {
			t.Errorf("Email = %q, want %q", principal.Email, "ana@example.com")
		}
		if len(principal.Roles) != 1 || principal.Roles[0] != "administrador" {
			t.Errorf("Roles = %v, want [administrador]", principal.Roles)
		}
	})

	t.Run("returns no roles when app_metadata role is absent", func(t *testing.T) {
		t.Parallel()

		verifier := newVerifier(t)
		token := signToken(t, key, jwt.MapClaims{
			"iss": issuer,
			"aud": "authenticated",
			"sub": "user-123",
			"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		})

		principal, err := verifier.Verify(context.Background(), token)
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}
		if principal.Roles != nil {
			t.Errorf("Roles = %v, want nil", principal.Roles)
		}
	})

	t.Run("rejects token signed with a different key", func(t *testing.T) {
		t.Parallel()

		verifier := newVerifier(t)
		token := signToken(t, otherKey, jwt.MapClaims{
			"iss": issuer,
			"aud": "authenticated",
			"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		})

		if _, err := verifier.Verify(context.Background(), token); err == nil {
			t.Error("Verify() error = nil, want error for token signed with unknown key")
		}
	})

	t.Run("rejects expired token", func(t *testing.T) {
		t.Parallel()

		verifier := newVerifier(t)
		token := signToken(t, key, jwt.MapClaims{
			"iss": issuer,
			"aud": "authenticated",
			"exp": jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		})

		if _, err := verifier.Verify(context.Background(), token); err == nil {
			t.Error("Verify() error = nil, want error for expired token")
		}
	})

	t.Run("rejects wrong issuer", func(t *testing.T) {
		t.Parallel()

		verifier := newVerifier(t)
		token := signToken(t, key, jwt.MapClaims{
			"iss": "https://another-project.supabase.co/auth/v1",
			"aud": "authenticated",
			"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		})

		if _, err := verifier.Verify(context.Background(), token); err == nil {
			t.Error("Verify() error = nil, want error for wrong issuer")
		}
	})

	t.Run("rejects wrong audience", func(t *testing.T) {
		t.Parallel()

		verifier := newVerifier(t)
		token := signToken(t, key, jwt.MapClaims{
			"iss": issuer,
			"aud": "anon",
			"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		})

		if _, err := verifier.Verify(context.Background(), token); err == nil {
			t.Error("Verify() error = nil, want error for wrong audience")
		}
	})

	t.Run("rejects unsigned token", func(t *testing.T) {
		t.Parallel()

		verifier := newVerifier(t)
		token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
			"iss": issuer,
			"aud": "authenticated",
			"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
		})
		signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		if err != nil {
			t.Fatalf("SignedString() error = %v", err)
		}

		if _, err := verifier.Verify(context.Background(), signed); err == nil {
			t.Error("Verify() error = nil, want error for alg=none token")
		}
	})
}

func TestNewVerifierFailsWhenJWKSUnavailable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	if _, err := supabase.NewVerifier(context.Background(), server.URL); err == nil {
		t.Error("NewVerifier() error = nil, want error when JWKS endpoint is unreachable")
	}
}

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}

	return key
}

func signToken(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKeyID

	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	return signed
}

func newJWKSServer(t *testing.T, key *rsa.PrivateKey) *httptest.Server {
	t.Helper()

	jwks := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": testKeyID,
				"n":   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /auth/v1/.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			t.Fatalf("encode jwks: %v", err)
		}
	})

	return httptest.NewServer(mux)
}
