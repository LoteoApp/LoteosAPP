package loteos

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

func authorizeEditor(
	ctx context.Context,
	repository gateway.LoteoRepository,
	actor Actor,
	loteoID string,
) error {
	if domain.HasRole(actor.Roles, domain.RolAdministrador) {
		return nil
	}
	if !domain.HasRole(actor.Roles, domain.RolAgrimensor) {
		return domain.ErrNoAutorizado
	}

	assigned, err := repository.IsAssignedToLoteo(ctx, actor.AuthProviderID, loteoID)
	if err != nil {
		return fromRepository(err)
	}
	if !assigned {
		return domain.ErrNoAutorizado
	}

	return nil
}
