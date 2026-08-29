package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// UpdateLote sets the values a lot can only get by hand: the DXF layers hold
// geometry with no text, so the lot number, price, area and description are
// loaded after the plan is on screen.
type UpdateLote interface {
	Execute(ctx context.Context, actor Actor, loteoID, loteID string, data domain.LoteData) (domain.Lote, error)
}

type updateLoteUseCase struct {
	repository gateway.LoteoRepository
}

func NewUpdateLote(repository gateway.LoteoRepository) UpdateLote {
	return &updateLoteUseCase{repository: repository}
}

// Execute authorizes before it validates, so a caller with no permission on
// the loteo learns nothing about which values the endpoint would accept.
func (useCase *updateLoteUseCase) Execute(
	ctx context.Context,
	actor Actor,
	loteoID, loteID string,
	data domain.LoteData,
) (domain.Lote, error) {
	if err := useCase.authorize(ctx, actor, loteoID); err != nil {
		return domain.Lote{}, err
	}

	data.Number = strings.TrimSpace(data.Number)
	data.Currency = strings.ToUpper(strings.TrimSpace(data.Currency))
	data.Features = strings.TrimSpace(data.Features)
	if err := data.Validate(); err != nil {
		return domain.Lote{}, err
	}

	lote, err := useCase.repository.UpdateLote(ctx, actor.AuthProviderID, loteoID, loteID, data)
	if err != nil {
		return domain.Lote{}, fromRepository(err)
	}

	return lote, nil
}

// authorize lets an administrador edit any loteo and an agrimensor only the
// ones assigned to them. Every other role is rejected without touching the
// repository.
func (useCase *updateLoteUseCase) authorize(ctx context.Context, actor Actor, loteoID string) error {
	if domain.HasRole(actor.Roles, domain.RolAdministrador) {
		return nil
	}
	if !domain.HasRole(actor.Roles, domain.RolAgrimensor) {
		return domain.ErrNoAutorizado
	}

	assigned, err := useCase.repository.IsAssignedToLoteo(ctx, actor.AuthProviderID, loteoID)
	if err != nil {
		return fromRepository(err)
	}
	if !assigned {
		return domain.ErrNoAutorizado
	}

	return nil
}
