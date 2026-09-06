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
	// State, when set, keeps only the lotes in that state. A sale form asks
	// for domain.LotStateAvailable so it never offers a lote that is already
	// reserved or sold.
	State domain.LotState
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
	if input.State != "" && !input.State.IsValid() {
		return nil, domain.ErrInvalidLotState
	}

	scope, err := loteoVisibility(input.Actor)
	if err != nil {
		return nil, err
	}

	filter := gateway.LoteFilter{Search: strings.TrimSpace(input.Search), State: input.State}
	lotes, err := useCase.repository.SearchLotes(ctx, filter, scope)
	if err != nil {
		return nil, fromRepository(err)
	}

	return lotes, nil
}
