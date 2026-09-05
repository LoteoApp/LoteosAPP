package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ListLoteArchivos returns the active fotos/planos attached to one lote. It
// applies the same visibility rules as GetLoteo: a loteo the actor may not
// see is reported as domain.ErrLoteoNotFound.
type ListLoteArchivos interface {
	Execute(ctx context.Context, actor Actor, loteoID, loteID string) ([]domain.Archivo, error)
}

type listLoteArchivosUseCase struct {
	repository gateway.LoteoRepository
}

func NewListLoteArchivos(repository gateway.LoteoRepository) ListLoteArchivos {
	return &listLoteArchivosUseCase{repository: repository}
}

func (useCase *listLoteArchivosUseCase) Execute(
	ctx context.Context,
	actor Actor,
	loteoID, loteID string,
) ([]domain.Archivo, error) {
	scope, err := loteoVisibility(actor)
	if err != nil {
		return nil, err
	}

	loteoID = strings.TrimSpace(loteoID)
	if loteoID == "" {
		return nil, domain.ErrLoteoNotFound
	}

	if _, err := useCase.repository.Get(ctx, loteoID, scope); err != nil {
		return nil, fromRepository(err)
	}

	archivos, err := useCase.repository.ListLoteArchivos(ctx, loteoID, loteID)
	if err != nil {
		return nil, fromRepository(err)
	}

	return archivos, nil
}
