package agencies

import (
	"context"
	"errors"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// resolveActorID maps the identity provider's subject to the usuarios.id
// stored as usuario_modificacion on every inmobiliaria write.
func resolveActorID(ctx context.Context, users gateway.UserRepository, subject string) (string, error) {
	actor, err := users.FindByAuthProviderID(ctx, subject)
	if err != nil {
		if errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			return "", domain.ErrActorNoAprovisionado
		}
		return "", err
	}

	return actor.ID, nil
}
