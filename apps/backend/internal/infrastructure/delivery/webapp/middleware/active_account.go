package middleware

import (
	"context"
	"errors"
	"net/http"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

type accountRepository interface {
	FindByAuthProviderID(ctx context.Context, authProviderID string) (domain.Usuario, error)
}

// RequireActiveAccount returns a middleware that blocks every request from a
// caller whose usuarios row is given de baja, so a baja takes effect right
// away instead of waiting for the caller's still-valid access token to
// expire — RequireAuth's token check alone can't see that, since it verifies
// the token's signature locally and never asks Supabase whether the account
// is still active. Must run behind RequireAuth, which populates the
// Principal this reads.
//
// A caller with no usuarios row at all (e.g. an administrador provisioned
// only in the identity provider) passes through unchanged: whether that's
// allowed is for each use case to decide, via domain.ErrActorNoAprovisionado.
func RequireActiveAccount(repository accountRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			principal, ok := PrincipalFromContext(request.Context())
			if !ok {
				unauthorized(w)
				return
			}

			usuario, err := repository.FindByAuthProviderID(request.Context(), principal.Subject)
			if err != nil {
				if errors.Is(err, domain.ErrUsuarioNoEncontrado) {
					next.ServeHTTP(w, request)
					return
				}
				response.WriteError(w, request, "active account lookup failed", err)
				return
			}

			if !usuario.Activo() {
				response.WriteError(w, request, "inactive account blocked", domain.ErrCuentaInactiva)
				return
			}

			next.ServeHTTP(w, request)
		})
	}
}
