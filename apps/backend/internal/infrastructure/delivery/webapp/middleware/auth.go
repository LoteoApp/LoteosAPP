package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"loteosapp/backend/internal/infrastructure/auth/keycloak"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type verifier interface {
	Verify(ctx context.Context, token string) (keycloak.Principal, error)
}

type contextKey struct{}

var principalContextKey = contextKey{}

// RequireAuth returns a middleware that rejects requests without a valid
// Keycloak bearer token and exposes the resulting Principal via
// PrincipalFromContext.
func RequireAuth(v verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			token, ok := bearerToken(request)
			if !ok {
				unauthorized(w)
				return
			}

			principal, err := v.Verify(request.Context(), token)
			if err != nil {
				slog.ErrorContext(request.Context(), "token verification failed", "error", err)
				unauthorized(w)
				return
			}

			ctx := context.WithValue(request.Context(), principalContextKey, principal)
			next.ServeHTTP(w, request.WithContext(ctx))
		})
	}
}

// PrincipalFromContext returns the Principal stored by RequireAuth, if any.
func PrincipalFromContext(ctx context.Context) (keycloak.Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(keycloak.Principal)
	return principal, ok
}

func bearerToken(request *http.Request) (string, bool) {
	header := request.Header.Get("Authorization")

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}

	token := strings.TrimPrefix(header, prefix)
	if token == "" {
		return "", false
	}

	return token, true
}

func unauthorized(w http.ResponseWriter) {
	response.WriteJSON(w, http.StatusUnauthorized, response.ErrorResponse{
		Code:    "unauthorized",
		Message: "No autorizado",
	})
}
