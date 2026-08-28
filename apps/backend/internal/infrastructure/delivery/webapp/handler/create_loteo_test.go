package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/usecase/loteos"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type createLoteoStub struct {
	loteo    domain.Loteo
	err      error
	called   bool
	gotActor loteos.Actor
	gotInput loteos.LoteoInput
}

func (stub *createLoteoStub) Execute(_ context.Context, actor loteos.Actor, input loteos.LoteoInput) (domain.Loteo, error) {
	stub.called = true
	stub.gotActor = actor
	stub.gotInput = input
	return stub.loteo, stub.err
}

func performCreateLoteoRequest(t *testing.T, createLoteo *createLoteoStub, verifier userVerifierStub, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	h := handler.NewCreateLoteoHandler(createLoteo)
	requireAuth := middleware.RequireAuth(verifier)

	mux := http.NewServeMux()
	mux.Handle("POST /api/v1/loteos", requireAuth(handler.Adapt(h, 5*time.Second)))

	return performRequest(t, mux, http.MethodPost, "/api/v1/loteos", token, body)
}

func administradorVerifier() userVerifierStub {
	return userVerifierStub{principal: supabase.Principal{
		Subject: "admin-1",
		Roles:   []string{domain.RolAdministrador},
	}}
}

func planoBody() map[string]any {
	square := []map[string]float64{
		{"x": 0, "y": 0}, {"x": 10, "y": 0}, {"x": 10, "y": 10}, {"x": 0, "y": 10},
	}

	return map[string]any{
		"nombre":      "Loteo Norte",
		"ubicacion":   "Córdoba",
		"descripcion": "Al norte de la ciudad",
		"plano": map[string]any{
			"loteo": map[string]any{"handle": "1A", "vertices": square},
			"manzanas": []map[string]any{
				{"ref": "MANZANA-0", "handle": "2A", "vertices": square},
			},
			"lotes": []map[string]any{
				{"manzanaRef": "MANZANA-0", "handle": "3A", "vertices": square},
			},
			"calles": []map[string]any{
				{"handle": "4A", "vertices": square},
			},
		},
	}
}

func TestCreateLoteoRoute(t *testing.T) {
	t.Parallel()

	t.Run("creates a loteo and answers with its ids", func(t *testing.T) {
		t.Parallel()

		createLoteo := &createLoteoStub{loteo: domain.Loteo{
			ID:       "loteo-1",
			Name:     "Loteo Norte",
			Manzanas: []domain.Manzana{{ID: "manzana-1"}},
			Lotes:    []domain.Lote{{ID: "lote-1", ManzanaID: "manzana-1"}},
			Calles:   []domain.Calle{{ID: "calle-1"}},
		}}

		recorder := performCreateLoteoRequest(t, createLoteo, administradorVerifier(), "valid-token", planoBody())

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusCreated, recorder.Body.String())
		}

		var got domain.Loteo
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.ID != "loteo-1" {
			t.Errorf("id = %q, want loteo-1", got.ID)
		}
		if len(got.Lotes) != 1 || got.Lotes[0].ManzanaID != "manzana-1" {
			t.Errorf("response lotes = %#v", got.Lotes)
		}
	})

	t.Run("passes the authenticated caller to the use case", func(t *testing.T) {
		t.Parallel()

		createLoteo := &createLoteoStub{}
		recorder := performCreateLoteoRequest(t, createLoteo, administradorVerifier(), "valid-token", planoBody())

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		if createLoteo.gotActor.AuthProviderID != "admin-1" {
			t.Errorf("actor id = %q, want the token subject", createLoteo.gotActor.AuthProviderID)
		}
		if !domain.HasRole(createLoteo.gotActor.Roles, domain.RolAdministrador) {
			t.Errorf("actor roles = %v, want the token roles", createLoteo.gotActor.Roles)
		}
	})

	t.Run("maps the plan onto the use case input", func(t *testing.T) {
		t.Parallel()

		createLoteo := &createLoteoStub{}
		recorder := performCreateLoteoRequest(t, createLoteo, administradorVerifier(), "valid-token", planoBody())

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}

		input := createLoteo.gotInput
		if input.Name != "Loteo Norte" || input.Location != "Córdoba" {
			t.Errorf("input = %#v", input)
		}
		if input.Plan == nil {
			t.Fatal("the plan should reach the use case")
		}
		if len(input.Plan.Loteo.Polygon) != 4 {
			t.Errorf("loteo ring = %d vertices, want 4", len(input.Plan.Loteo.Polygon))
		}
		if len(input.Plan.Manzanas) != 1 || input.Plan.Manzanas[0].Ref != "MANZANA-0" {
			t.Errorf("manzanas = %#v", input.Plan.Manzanas)
		}
		if len(input.Plan.Lotes) != 1 || input.Plan.Lotes[0].ManzanaRef != "MANZANA-0" {
			t.Errorf("lotes = %#v", input.Plan.Lotes)
		}
		if len(input.Plan.Calles) != 1 || input.Plan.Calles[0].Handle != "4A" {
			t.Errorf("calles = %#v", input.Plan.Calles)
		}
	})

	t.Run("accepts a loteo registered without a plan", func(t *testing.T) {
		t.Parallel()

		createLoteo := &createLoteoStub{}
		recorder := performCreateLoteoRequest(t, createLoteo, administradorVerifier(), "valid-token",
			map[string]any{"nombre": "Loteo Norte"})

		if recorder.Code != http.StatusCreated {
			t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
		}
		if createLoteo.gotInput.Plan != nil {
			t.Error("a body without a plan should reach the use case with a nil plan")
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		createLoteo := &createLoteoStub{}
		recorder := performCreateLoteoRequest(t, createLoteo, userVerifierStub{}, "", planoBody())

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if createLoteo.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("rejects an invalid JSON body", func(t *testing.T) {
		t.Parallel()

		createLoteo := &createLoteoStub{}
		recorder := performCreateLoteoRequest(t, createLoteo, administradorVerifier(), "valid-token", "not-json")

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if createLoteo.called {
			t.Error("use case should not be called with an invalid body")
		}
	})

	t.Run("rejects a body past the size limit", func(t *testing.T) {
		t.Parallel()

		// Valid JSON, but far larger than any plan the endpoint accepts: the
		// decoder must stop reading instead of allocating the whole body.
		oversized := `{"nombre":"` + strings.Repeat("a", 17<<20) + `"}`

		createLoteo := &createLoteoStub{}
		recorder := performCreateLoteoRequest(t, createLoteo, administradorVerifier(), "valid-token", oversized)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
		if createLoteo.called {
			t.Error("use case should not be called with an oversized body")
		}
	})

	t.Run("maps use case errors to the right status", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			err        error
			wantStatus int
			wantCode   string
		}{
			{name: "not authorized", err: domain.ErrNoAutorizado, wantStatus: http.StatusForbidden, wantCode: "forbidden"},
			{name: "missing name", err: domain.ErrInvalidLoteoName, wantStatus: http.StatusBadRequest, wantCode: "invalid_loteo_nombre"},
			{name: "invalid geometry", err: domain.ErrInvalidGeometry, wantStatus: http.StatusBadRequest, wantCode: "invalid_geometry"},
			{name: "unknown manzana", err: domain.ErrUnknownManzana, wantStatus: http.StatusBadRequest, wantCode: "unknown_manzana"},
			{name: "plan too large", err: domain.ErrPlanTooLarge, wantStatus: http.StatusBadRequest, wantCode: "plan_too_large"},
			{name: "unexpected error", err: errors.New("connection refused"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				t.Parallel()

				createLoteo := &createLoteoStub{err: test.err}
				recorder := performCreateLoteoRequest(t, createLoteo, administradorVerifier(), "valid-token", planoBody())

				if recorder.Code != test.wantStatus {
					t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
				}

				var got response.ErrorResponse
				if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
					t.Fatalf("decode response: %v", err)
				}
				if got.Code != test.wantCode {
					t.Errorf("error code = %q, want %q", got.Code, test.wantCode)
				}
				if test.wantStatus == http.StatusInternalServerError && got.Message == test.err.Error() {
					t.Error("response exposes the internal error")
				}
			})
		}
	})
}
