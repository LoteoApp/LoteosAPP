package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ListLoteoFiles returns the active fotos/planos attached to a loteo
// itself. It applies the same visibility rules as GetLoteo: a loteo the
// actor may not see is reported as domain.ErrLoteoNotFound.
type ListLoteoFiles interface {
	Execute(ctx context.Context, actor Actor, loteoID string) ([]domain.File, error)
}

type listLoteoFilesUseCase struct {
	repository gateway.LoteoRepository
}

func NewListLoteoFiles(repository gateway.LoteoRepository) ListLoteoFiles {
	return &listLoteoFilesUseCase{repository: repository}
}

func (useCase *listLoteoFilesUseCase) Execute(
	ctx context.Context,
	actor Actor,
	loteoID string,
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

	files, err := useCase.repository.ListLoteoFiles(ctx, loteoID)
	if err != nil {
		return nil, fromRepository(err)
	}

	return files, nil
}
