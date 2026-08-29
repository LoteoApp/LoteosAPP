package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"loteosapp/backend/internal/infrastructure/delivery/webapp/server"
)

const frontendOrigin = "http://localhost:5173"

func okHandler(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*called = true
		w.WriteHeader(http.StatusOK)
	})
}

func TestWithCORS(t *testing.T) {
	t.Parallel()

	t.Run("answers the configured origin with the CORS headers", func(t *testing.T) {
		t.Parallel()

		var called bool
		request := httptest.NewRequest(http.MethodGet, "/api/v1/loteos", nil)
		request.Header.Set("Origin", frontendOrigin)
		recorder := httptest.NewRecorder()

		server.WithCORS(frontendOrigin, okHandler(&called)).ServeHTTP(recorder, request)

		if !called {
			t.Error("the request should reach the wrapped handler")
		}
		if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != frontendOrigin {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, frontendOrigin)
		}
		if recorder.Header().Get("Access-Control-Allow-Headers") == "" ||
			recorder.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("an allowed origin should also learn the accepted headers and methods")
		}
	})

	t.Run("does not allow another origin", func(t *testing.T) {
		t.Parallel()

		var called bool
		request := httptest.NewRequest(http.MethodGet, "/api/v1/loteos", nil)
		request.Header.Set("Origin", "http://evil.example")
		recorder := httptest.NewRecorder()

		server.WithCORS(frontendOrigin, okHandler(&called)).ServeHTTP(recorder, request)

		if recorder.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("an origin that isn't the configured one must not be allowed")
		}
		// The response still reaches the handler: it's the browser, not the
		// server, that stops the caller from reading it.
		if !called {
			t.Error("the request should still reach the wrapped handler")
		}
	})

	t.Run("answers a preflight without reaching the handler", func(t *testing.T) {
		t.Parallel()

		var called bool
		request := httptest.NewRequest(http.MethodOptions, "/api/v1/loteos", nil)
		request.Header.Set("Origin", frontendOrigin)
		recorder := httptest.NewRecorder()

		server.WithCORS(frontendOrigin, okHandler(&called)).ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNoContent {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusNoContent)
		}
		if called {
			t.Error("a preflight should not reach the wrapped handler")
		}
	})

	t.Run("varies on Origin even without one", func(t *testing.T) {
		t.Parallel()

		var called bool
		request := httptest.NewRequest(http.MethodGet, "/api/v1/loteos", nil)
		recorder := httptest.NewRecorder()

		server.WithCORS(frontendOrigin, okHandler(&called)).ServeHTTP(recorder, request)

		// Without it a cache could serve one origin's response to another.
		if recorder.Header().Get("Vary") != "Origin" {
			t.Errorf("Vary = %q, want Origin", recorder.Header().Get("Vary"))
		}
	})
}
