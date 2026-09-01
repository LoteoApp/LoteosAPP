package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type ListLoteosInput struct {
	Actor  Actor
	Search string
}

// ListLoteos returns the loteos the actor is allowed to see, as summaries
// without geometry. administrador and administrativo see all of them;
// agrimensor, escribano and inmobiliaria see only their assigned loteos.
type ListLoteos interface {
	Execute(ctx context.Context, input ListLoteosInput) ([]domain.LoteoSummary, error)
}

type listLoteosUseCase struct {
	repository gateway.LoteoRepository
}

func NewListLoteos(repository gateway.LoteoRepository) ListLoteos {
	return &listLoteosUseCase{repository: repository}
}

func (useCase *listLoteosUseCase) Execute(
	ctx context.Context,
	input ListLoteosInput,
) ([]domain.LoteoSummary, error) {
	scope, err := loteoVisibility(input.Actor)
	if err != nil {
		return nil, err
	}

	loteos, err := useCase.repository.List(ctx, strings.TrimSpace(input.Search), scope)
	if err != nil {
		return nil, fromRepository(err)
	}

	return loteos, nil
}
