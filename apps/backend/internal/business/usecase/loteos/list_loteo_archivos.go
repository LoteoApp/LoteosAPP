package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ListLoteoArchivos returns the active fotos/planos attached to a loteo
// itself. It applies the same visibility rules as GetLoteo: a loteo the
// actor may not see is reported as domain.ErrLoteoNotFound.
type ListLoteoArchivos interface {
	Execute(ctx context.Context, actor Actor, loteoID string) ([]domain.Archivo, error)
}

type listLoteoArchivosUseCase struct {
	repository gateway.LoteoRepository
}

func NewListLoteoArchivos(repository gateway.LoteoRepository) ListLoteoArchivos {
	return &listLoteoArchivosUseCase{repository: repository}
}

func (useCase *listLoteoArchivosUseCase) Execute(
	ctx context.Context,
	actor Actor,
	loteoID string,
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

	archivos, err := useCase.repository.ListLoteoArchivos(ctx, loteoID)
	if err != nil {
		return nil, fromRepository(err)
	}

	return archivos, nil
}
