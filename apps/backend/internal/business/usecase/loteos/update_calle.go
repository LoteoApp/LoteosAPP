package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type UpdateCalle interface {
	Execute(ctx context.Context, actor Actor, loteoID, calleID string, data domain.CalleData) (domain.Calle, error)
}

type updateCalleUseCase struct {
	repository gateway.LoteoRepository
}

func NewUpdateCalle(repository gateway.LoteoRepository) UpdateCalle {
	return &updateCalleUseCase{repository: repository}
}

func (useCase *updateCalleUseCase) Execute(
	ctx context.Context,
	actor Actor,
	loteoID, calleID string,
	data domain.CalleData,
) (domain.Calle, error) {
	if err := authorizeEditor(ctx, useCase.repository, actor, loteoID); err != nil {
		return domain.Calle{}, err
	}

	data.Name = strings.TrimSpace(data.Name)
	data.Type = strings.ToLower(strings.TrimSpace(data.Type))
	if err := data.Validate(); err != nil {
		return domain.Calle{}, err
	}

	calle, err := useCase.repository.UpdateCalle(ctx, actor.AuthProviderID, loteoID, calleID, data)
	if err != nil {
		return domain.Calle{}, fromRepository(err)
	}

	return calle, nil
}
