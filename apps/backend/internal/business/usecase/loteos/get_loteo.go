package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// GetLoteo returns one loteo with its manzanas, lotes, calles and geometry.
// It applies the same visibility rules as ListLoteos; a loteo the actor may
// not see is reported as domain.ErrLoteoNotFound rather than a forbidden, so
// the response doesn't reveal which ids exist.
type GetLoteo interface {
	Execute(ctx context.Context, actor Actor, loteoID string) (domain.Loteo, error)
}

type getLoteoUseCase struct {
	repository gateway.LoteoRepository
}

func NewGetLoteo(repository gateway.LoteoRepository) GetLoteo {
	return &getLoteoUseCase{repository: repository}
}

func (useCase *getLoteoUseCase) Execute(
	ctx context.Context,
	actor Actor,
	loteoID string,
) (domain.Loteo, error) {
	scope, err := loteoVisibility(actor)
	if err != nil {
		return domain.Loteo{}, err
	}

	loteoID = strings.TrimSpace(loteoID)
	if loteoID == "" {
		return domain.Loteo{}, domain.ErrLoteoNotFound
	}

	loteo, err := useCase.repository.Get(ctx, loteoID, scope)
	if err != nil {
		return domain.Loteo{}, fromRepository(err)
	}

	return loteo, nil
}
