package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ListLoteFiles returns the active fotos/planos attached to one lote. It
// applies the same visibility rules as GetLoteo: a loteo the actor may not
// see is reported as domain.ErrLoteoNotFound.
type ListLoteFiles interface {
	Execute(ctx context.Context, actor Actor, loteoID, loteID string) ([]domain.File, error)
}

type listLoteFilesUseCase struct {
	repository gateway.LoteoRepository
}

func NewListLoteFiles(repository gateway.LoteoRepository) ListLoteFiles {
	return &listLoteFilesUseCase{repository: repository}
}

func (useCase *listLoteFilesUseCase) Execute(
	ctx context.Context,
	actor Actor,
	loteoID, loteID string,
) ([]domain.File, error) {
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

	files, err := useCase.repository.ListLoteFiles(ctx, loteoID, loteID)
	if err != nil {
		return nil, fromRepository(err)
	}

	return files, nil
}
