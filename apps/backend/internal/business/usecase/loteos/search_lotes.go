package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type SearchLotesInput struct {
	Actor  Actor
	Search string
}

// SearchLotes returns the lotes the actor may see as summaries, so a caller
// can pick one to sell. It is restricted to the roles that carry a sale —
// administrador, administrativo and inmobiliaria — because the result
// carries the price of every lote. An inmobiliaria only reaches the lotes of
// the loteos assigned to its agency.
type SearchLotes interface {
	Execute(ctx context.Context, input SearchLotesInput) ([]domain.LoteSummary, error)
}

type searchLotesUseCase struct {
	repository gateway.LoteoRepository
}

func NewSearchLotes(repository gateway.LoteoRepository) SearchLotes {
	return &searchLotesUseCase{repository: repository}
}

func (useCase *searchLotesUseCase) Execute(
	ctx context.Context,
	input SearchLotesInput,
) ([]domain.LoteSummary, error) {
	if !domain.HasRole(input.Actor.Roles, domain.RolAdministrador) &&
		!domain.HasRole(input.Actor.Roles, domain.RolAdministrativo) &&
		!domain.HasRole(input.Actor.Roles, domain.RolInmobiliaria) {
		return nil, domain.ErrNoAutorizado
	}

	scope, err := loteoVisibility(input.Actor)
	if err != nil {
		return nil, err
	}

	lotes, err := useCase.repository.SearchLotes(ctx, strings.TrimSpace(input.Search), scope)
	if err != nil {
		return nil, fromRepository(err)
	}

	return lotes, nil
}
