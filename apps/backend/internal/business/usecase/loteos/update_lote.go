package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type UpdateLote interface {
	Execute(ctx context.Context, actor Actor, loteoID, loteID string, data domain.LoteData) (domain.Lote, error)
}

type updateLoteUseCase struct {
	repository gateway.LoteoRepository
}

func NewUpdateLote(repository gateway.LoteoRepository) UpdateLote {
	return &updateLoteUseCase{repository: repository}
}

func (useCase *updateLoteUseCase) Execute(
	ctx context.Context,
	actor Actor,
	loteoID, loteID string,
	data domain.LoteData,
) (domain.Lote, error) {
	if err := authorizeEditor(ctx, useCase.repository, actor, loteoID); err != nil {
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
