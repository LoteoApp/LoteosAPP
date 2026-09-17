package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRunHealthcheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("expected request to /health, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Setenv("PORT", portFromURL(t, server.URL))

	if got := runHealthcheck(); got != 0 {
		t.Errorf("runHealthcheck() = %d, want 0", got)
	}
}

func TestRunHealthcheck_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	t.Setenv("PORT", portFromURL(t, server.URL))

	if got := runHealthcheck(); got != 1 {
		t.Errorf("runHealthcheck() = %d, want 1", got)
	}
}

func TestRunHealthcheck_ConnectionFailure(t *testing.T) {
	t.Setenv("PORT", "1")

	if got := runHealthcheck(); got != 1 {
		t.Errorf("runHealthcheck() = %d, want 1", got)
	}
}

func portFromURL(t *testing.T, rawURL string) string {
	t.Helper()

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}

	return parsed.Port()
}
