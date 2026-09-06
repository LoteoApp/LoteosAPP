package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
)

type searchLotesStub struct {
	result   []domain.LoteSummary
	err      error
	called   bool
	gotInput loteos.SearchLotesInput
}

func (stub *searchLotesStub) Execute(_ context.Context, input loteos.SearchLotesInput) ([]domain.LoteSummary, error) {
	stub.called = true
	stub.gotInput = input
	return stub.result, stub.err
}

func performSearchLotesRequest(
	t *testing.T,
	searchLotes *searchLotesStub,
	verifier userVerifierStub,
	token, query string,
) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewSearchLotesHandler(searchLotes)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/lotes", requireAuth(handler.Adapt(h, 5*time.Second)))

	path := "/api/v1/lotes"
	if query != "" {
		path += "?" + query
	}
	return performRequest(t, mux, http.MethodGet, path, token, nil)
}

func TestSearchLotesRoute(t *testing.T) {
	t.Parallel()

	t.Run("lists lotes with their manzana and loteo for an authorized caller", func(t *testing.T) {
		t.Parallel()

		price := 150000.0
		area := 300.0
		searchLotes := &searchLotesStub{result: []domain.LoteSummary{{
			ID: "lote-1", Number: "7",
			ManzanaID: "mz-1", ManzanaNumber: "1",
			LoteoID: "loteo-1", LoteoName: "Norte",
			Price: &price, Currency: "USD", Area: &area,
		}}}
		verifier := userVerifierStub{principal: supabase.Principal{
			Subject: "user-1", Roles: []string{domain.RolAdministrativo},
		}}

		recorder := performSearchLotesRequest(t, searchLotes, verifier, "valid-token", "q=norte")

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if searchLotes.gotInput.Search != "norte" {
			t.Errorf("search passed to use case = %q, want %q", searchLotes.gotInput.Search, "norte")
		}
		if searchLotes.gotInput.Actor.AuthProviderID != "user-1" {
			t.Errorf("actor passed to use case = %#v", searchLotes.gotInput.Actor)
		}

		var got struct {
			Lotes []domain.LoteSummary `json:"lotes"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(got.Lotes) != 1 {
			t.Fatalf("response = %#v", got.Lotes)
		}
		lote := got.Lotes[0]
		if lote.ID != "lote-1" || lote.Number != "7" || lote.ManzanaNumber != "1" || lote.LoteoName != "Norte" {
			t.Errorf("response = %#v", lote)
		}
		if lote.Price == nil || *lote.Price != price {
			t.Errorf("precio = %v, want %v", lote.Price, price)
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		searchLotes := &searchLotesStub{}
		recorder := performSearchLotesRequest(t, searchLotes, userVerifierStub{}, "", "")

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if searchLotes.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("maps an unauthorized role to forbidden", func(t *testing.T) {
		t.Parallel()

		searchLotes := &searchLotesStub{err: domain.ErrNoAutorizado}
		verifier := userVerifierStub{principal: supabase.Principal{
			Subject: "user-1", Roles: []string{domain.RolAgrimensor},
		}}

		recorder := performSearchLotesRequest(t, searchLotes, verifier, "valid-token", "")

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
		}
	})

	t.Run("hides an unexpected error behind a 500", func(t *testing.T) {
		t.Parallel()

		searchLotes := &searchLotesStub{err: domain.ErrDatabaseUnavailable.WithCause(context.DeadlineExceeded)}
		verifier := userVerifierStub{principal: supabase.Principal{
			Subject: "user-1", Roles: []string{domain.RolAdministrador},
		}}

		recorder := performSearchLotesRequest(t, searchLotes, verifier, "valid-token", "")

		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
		}
	})
}
