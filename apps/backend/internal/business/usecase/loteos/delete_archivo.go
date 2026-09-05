package loteos

import (
	"context"

	"loteosapp/backend/internal/business/gateway"
)

// DeleteArchivo soft-deletes one active foto/plano, whether it's attached to
// the loteo itself or to one of its lotes. The object in storage is left in
// place, the same way replacing a DXF leaves the superseded one. Only an
// administrador, or an agrimensor assigned to the loteo, may do this.
type DeleteArchivo interface {
	Execute(ctx context.Context, actor Actor, loteoID, archivoID string) error
}

type deleteArchivoUseCase struct {
	repository gateway.LoteoRepository
}

func NewDeleteArchivo(repository gateway.LoteoRepository) DeleteArchivo {
	return &deleteArchivoUseCase{repository: repository}
}

func (useCase *deleteArchivoUseCase) Execute(ctx context.Context, actor Actor, loteoID, archivoID string) error {
	if err := authorizeEditor(ctx, useCase.repository, actor, loteoID); err != nil {
		return err
	}

	if err := useCase.repository.DeleteArchivo(ctx, actor.AuthProviderID, loteoID, archivoID); err != nil {
		return fromRepository(err)
	}

	return nil
}
