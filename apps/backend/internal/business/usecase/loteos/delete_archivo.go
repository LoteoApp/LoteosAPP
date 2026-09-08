package loteos

import (
	"context"

	"loteosapp/backend/internal/business/gateway"
)

// DeleteFile soft-deletes one active foto/plano, whether it's attached to
// the loteo itself or to one of its lotes. The object in storage is left in
// place, the same way replacing a DXF leaves the superseded one. Only an
// administrador, or an agrimensor assigned to the loteo, may do this.
type DeleteFile interface {
	Execute(ctx context.Context, actor Actor, loteoID, fileID string) error
}

type deleteFileUseCase struct {
	repository gateway.LoteoRepository
}

func NewDeleteFile(repository gateway.LoteoRepository) DeleteFile {
	return &deleteFileUseCase{repository: repository}
}

func (useCase *deleteFileUseCase) Execute(ctx context.Context, actor Actor, loteoID, fileID string) error {
	if err := authorizeEditor(ctx, useCase.repository, actor, loteoID); err != nil {
		return err
	}

	if err := useCase.repository.DeleteFile(ctx, actor.AuthProviderID, loteoID, fileID); err != nil {
		return fromRepository(err)
	}

	return nil
}
