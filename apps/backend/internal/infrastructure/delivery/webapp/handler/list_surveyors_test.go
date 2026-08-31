package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type listSurveyorsStub struct {
	agrimensores       []domain.Usuario
	err                error
	called             bool
	gotIncludeInactive bool
}

func (stub *listSurveyorsStub) Execute(_ context.Context, _ []string, includeInactive bool) ([]domain.Usuario, error) {
	stub.called = true
	stub.gotIncludeInactive = includeInactive
	return stub.agrimensores, stub.err
}

func performListSurveyorsRequest(t *testing.T, listSurveyors *listSurveyorsStub, verifier userVerifierStub, token, path string) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/agrimensores", middleware.RequireAuth(verifier)(
		handler.Adapt(handler.NewListSurveyorsHandler(listSurveyors), 5*time.Second)))

	return performRequest(t, mux, http.MethodGet, path, token, nil)
}

func TestListSurveyorsRoute(t *testing.T) {
	t.Parallel()

	t.Run("returns the agrimensores", func(t *testing.T) {
		t.Parallel()

		baja := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		listSurveyors := &listSurveyorsStub{agrimensores: []domain.Usuario{
			{ID: "agri-1", Nombre: "Ana", Apellido: "Gómez", Rol: domain.RolAgrimensor},
			{ID: "agri-2", Nombre: "Luis", Apellido: "Paz", Rol: domain.RolAgrimensor, FechaBaja: &baja},
		}}

		recorder := performListSurveyorsRequest(t, listSurveyors, administradorVerifier(), "valid-token", "/api/v1/agrimensores")

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
		}

		var got struct {
			Agrimensores []struct {
				ID        string  `json:"id"`
				FechaBaja *string `json:"fechaBaja"`
			} `json:"agrimensores"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(got.Agrimensores) != 2 {
			t.Fatalf("agrimensores = %d, want 2", len(got.Agrimensores))
		}
		if got.Agrimensores[0].FechaBaja != nil {
			t.Error("an active agrimensor should carry a null fechaBaja")
		}
		if got.Agrimensores[1].FechaBaja == nil {
			t.Error("an agrimensor given de baja should carry its fechaBaja")
		}
	})

	t.Run("serializes an empty list as an array", func(t *testing.T) {
		t.Parallel()

		listSurveyors := &listSurveyorsStub{agrimensores: []domain.Usuario{}}

		recorder := performListSurveyorsRequest(t, listSurveyors, administradorVerifier(), "valid-token", "/api/v1/agrimensores")

		if body := recorder.Body.String(); body != "{\"agrimensores\":[]}\n" && body != "{\"agrimensores\":[]}" {
			t.Errorf("body = %q, want an empty array", body)
		}
	})

	t.Run("asks for the inactive agrimensores only when requested", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			path string
			want bool
		}{
			{path: "/api/v1/agrimensores", want: false},
			{path: "/api/v1/agrimensores?incluirBajas=false", want: false},
			{path: "/api/v1/agrimensores?incluirBajas=true", want: true},
		}

		for _, test := range tests {
			t.Run(test.path, func(t *testing.T) {
				t.Parallel()

				listSurveyors := &listSurveyorsStub{}
				performListSurveyorsRequest(t, listSurveyors, administradorVerifier(), "valid-token", test.path)

				if listSurveyors.gotIncludeInactive != test.want {
					t.Errorf("includeInactive = %v, want %v", listSurveyors.gotIncludeInactive, test.want)
				}
			})
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		listSurveyors := &listSurveyorsStub{}
		recorder := performListSurveyorsRequest(t, listSurveyors, userVerifierStub{}, "", "/api/v1/agrimensores")

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if listSurveyors.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("maps a forbidden use case error to 403", func(t *testing.T) {
		t.Parallel()

		listSurveyors := &listSurveyorsStub{err: domain.ErrNoAutorizado}
		recorder := performListSurveyorsRequest(t, listSurveyors, administradorVerifier(), "valid-token", "/api/v1/agrimensores")

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
		}

		var got response.ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.Code != "forbidden" {
			t.Errorf("error code = %q, want %q", got.Code, "forbidden")
		}
	})
}
