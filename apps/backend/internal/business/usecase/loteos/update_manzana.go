package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type UpdateManzana interface {
	Execute(ctx context.Context, actor Actor, loteoID, manzanaID string, data domain.ManzanaData) (domain.Manzana, error)
}

type updateManzanaUseCase struct {
	repository gateway.LoteoRepository
}

func NewUpdateManzana(repository gateway.LoteoRepository) UpdateManzana {
	return &updateManzanaUseCase{repository: repository}
}

func (useCase *updateManzanaUseCase) Execute(
	ctx context.Context,
	actor Actor,
	loteoID, manzanaID string,
	data domain.ManzanaData,
) (domain.Manzana, error) {
	if err := authorizeEditor(ctx, useCase.repository, actor, loteoID); err != nil {
		return domain.Manzana{}, err
	}

	data.Number = strings.TrimSpace(data.Number)
	data.CalleIDs = normalizeCalleIDs(data.CalleIDs)
	if err := data.Validate(); err != nil {
		return domain.Manzana{}, err
	}

	manzana, err := useCase.repository.UpdateManzana(ctx, actor.AuthProviderID, loteoID, manzanaID, data)
	if err != nil {
		return domain.Manzana{}, fromRepository(err)
	}

	return manzana, nil
}

func normalizeCalleIDs(ids []string) []string {
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		normalized = append(normalized, id)
	}
	return normalized
}
