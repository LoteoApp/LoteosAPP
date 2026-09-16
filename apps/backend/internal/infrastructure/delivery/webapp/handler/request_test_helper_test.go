package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"loteosapp/backend/internal/infrastructure/auth/supabase"
)

type userVerifierStub struct {
	principal supabase.Principal
	err       error
}

func (stub userVerifierStub) Verify(context.Context, string) (supabase.Principal, error) {
	return stub.principal, stub.err
}

func performRequest(t *testing.T, mux *http.ServeMux, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	switch v := body.(type) {
	case nil:
		reader = bytes.NewReader(nil)
	case string:
		reader = bytes.NewReader([]byte(v))
	default:
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}

	request := httptest.NewRequest(method, path, reader)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	return recorder
}
