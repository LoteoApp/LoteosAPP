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

type listUsersStub struct {
	usuarios           []domain.Usuario
	err                error
	called             bool
	gotIncludeInactive bool
}

func (stub *listUsersStub) Execute(_ context.Context, _ []string, includeInactive bool) ([]domain.Usuario, error) {
	stub.called = true
	stub.gotIncludeInactive = includeInactive
	return stub.usuarios, stub.err
}

func performListUsersRequest(t *testing.T, listUsers *listUsersStub, verifier userVerifierStub, token, path string) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	mux.Handle("GET /api/v1/usuarios", middleware.RequireAuth(verifier)(
		handler.Adapt(handler.NewListUsersHandler(listUsers), 5*time.Second)))

	return performRequest(t, mux, http.MethodGet, path, token, nil)
}

func TestListUsersRoute(t *testing.T) {
	t.Parallel()

	t.Run("returns the users", func(t *testing.T) {
		t.Parallel()

		baja := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		listUsers := &listUsersStub{usuarios: []domain.Usuario{
			{ID: "user-1", Nombre: "Ana", Apellido: "Gómez", Rol: domain.RolEscribano},
			{ID: "user-2", Nombre: "Luis", Apellido: "Paz", Rol: domain.RolAdministrativo, FechaBaja: &baja},
		}}

		recorder := performListUsersRequest(t, listUsers, administradorVerifier(), "valid-token", "/api/v1/usuarios")

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d, body = %s", recorder.Code, http.StatusOK, recorder.Body)
		}

		var got struct {
			Usuarios []struct {
				ID        string  `json:"id"`
				FechaBaja *string `json:"fechaBaja"`
			} `json:"usuarios"`
		}
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(got.Usuarios) != 2 {
			t.Fatalf("usuarios = %d, want 2", len(got.Usuarios))
		}
		if got.Usuarios[0].FechaBaja != nil {
			t.Error("an active user should carry a null fechaBaja")
		}
		if got.Usuarios[1].FechaBaja == nil {
			t.Error("a user given de baja should carry its fechaBaja")
		}
	})

	t.Run("serializes an empty list as an array", func(t *testing.T) {
		t.Parallel()

		listUsers := &listUsersStub{usuarios: []domain.Usuario{}}

		recorder := performListUsersRequest(t, listUsers, administradorVerifier(), "valid-token", "/api/v1/usuarios")

		if body := recorder.Body.String(); body != "{\"usuarios\":[]}\n" && body != "{\"usuarios\":[]}" {
			t.Errorf("body = %q, want an empty array", body)
		}
	})

	t.Run("asks for the inactive users only when requested", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			path string
			want bool
		}{
			{path: "/api/v1/usuarios", want: false},
			{path: "/api/v1/usuarios?incluirBajas=false", want: false},
			{path: "/api/v1/usuarios?incluirBajas=true", want: true},
		}

		for _, test := range tests {
			t.Run(test.path, func(t *testing.T) {
				t.Parallel()

				listUsers := &listUsersStub{}
				performListUsersRequest(t, listUsers, administradorVerifier(), "valid-token", test.path)

				if listUsers.gotIncludeInactive != test.want {
					t.Errorf("includeInactive = %v, want %v", listUsers.gotIncludeInactive, test.want)
				}
			})
		}
	})

	t.Run("rejects requests without a token", func(t *testing.T) {
		t.Parallel()

		listUsers := &listUsersStub{}
		recorder := performListUsersRequest(t, listUsers, userVerifierStub{}, "", "/api/v1/usuarios")

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
		}
		if listUsers.called {
			t.Error("use case should not be called without a token")
		}
	})

	t.Run("maps a forbidden use case error to 403", func(t *testing.T) {
		t.Parallel()

		listUsers := &listUsersStub{err: domain.ErrNoAutorizado}
		recorder := performListUsersRequest(t, listUsers, administradorVerifier(), "valid-token", "/api/v1/usuarios")

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
		}

		var got response.ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got.Code != "forbidden" {
			t.Errorf("error code = %q, want %q", got.Code, "forbidden")
		}
	})
}
