package keycloak_test

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

	"loteosapp/backend/internal/infrastructure/auth/keycloak"
)

const (
	testAudience = "loteosapp-backend"
	testKeyID    = "test-key"
)

func TestVerifierVerify(t *testing.T) {
	t.Parallel()

	key, otherKey := generateRSAKey(t), generateRSAKey(t)
	server := newJWKSServer(t, key)
	t.Cleanup(server.Close)

	newVerifier := func(t *testing.T) *keycloak.Verifier {
		t.Helper()

		verifier, err := keycloak.NewVerifier(context.Background(), server.URL, server.URL, testAudience)
		if err != nil {
			t.Fatalf("NewVerifier() error = %v", err)
		}
		t.Cleanup(verifier.Close)

		return verifier
	}

	t.Run("valid token merges realm and client roles", func(t *testing.T) {
		t.Parallel()

		verifier := newVerifier(t)
		token := signToken(t, key, jwt.MapClaims{
			"iss": server.URL,
			"aud": testAudience,
			"sub": "user-123",
			"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
			"realm_access": map[string]any{
				"roles": []string{"administrador"},
			},
			"resource_access": map[string]any{
				testAudience: map[string]any{
					"roles": []string{"administrador", "loteos:read"},
				},
			},
		})

		principal, err := verifier.Verify(context.Background(), token)
		if err != nil {
			t.Fatalf("Verify() error = %v", err)
		}

		if principal.Subject != "user-123" {
			t.Errorf("Subject = %q, want %q", principal.Subject, "user-123")
		}
		if len(principal.Roles) != 2 || principal.Roles[0] != "administrador" || principal.Roles[1] != "loteos:read" {
			t.Errorf("Roles = %v, want [administrador loteos:read] (deduplicated)", principal.Roles)
		}
	})

	t.Run("rejects token signed with a different key", func(t *testing.T) {
		t.Parallel()

		verifier := newVerifier(t)
		token := signToken(t, otherKey, jwt.MapClaims{
			"iss": server.URL,
			"aud": testAudience,
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
			"iss": server.URL,
			"aud": testAudience,
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
			"iss": "another-issuer",
			"aud": testAudience,
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
			"iss": server.URL,
			"aud": "another-audience",
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
			"iss": server.URL,
			"aud": testAudience,
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

// TestVerifierVerifyAcceptsIssuerDifferentFromJWKSURL covers split-horizon
// DNS setups (e.g. Docker Compose): the JWKS must be fetched from a URL
// reachable by this process, but tokens can carry a different, canonical
// `iss` (governed by Keycloak's KC_HOSTNAME) that callers reach through a
// different, publicly reachable URL.
func TestVerifierVerifyAcceptsIssuerDifferentFromJWKSURL(t *testing.T) {
	t.Parallel()

	key := generateRSAKey(t)
	jwksServer := newJWKSServer(t, key)
	t.Cleanup(jwksServer.Close)

	const canonicalIssuer = "http://localhost:8081/realms/loteosapp"

	verifier, err := keycloak.NewVerifier(context.Background(), jwksServer.URL, canonicalIssuer, testAudience)
	if err != nil {
		t.Fatalf("NewVerifier() error = %v", err)
	}
	t.Cleanup(verifier.Close)

	token := signToken(t, key, jwt.MapClaims{
		"iss": canonicalIssuer,
		"aud": testAudience,
		"sub": "user-123",
		"exp": jwt.NewNumericDate(time.Now().Add(time.Hour)),
	})

	if _, err := verifier.Verify(context.Background(), token); err != nil {
		t.Fatalf("Verify() error = %v, want a token issued for the canonical issuer to be accepted", err)
	}
}

func TestNewVerifierFailsWhenJWKSUnavailable(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	if _, err := keycloak.NewVerifier(context.Background(), server.URL, server.URL, testAudience); err == nil {
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
	mux.HandleFunc("GET /protocol/openid-connect/certs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(jwks); err != nil {
			t.Fatalf("encode jwks: %v", err)
		}
	})

	return httptest.NewServer(mux)
}
