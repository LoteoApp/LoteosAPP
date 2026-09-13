package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

type AgencyRepository interface {
	Create(ctx context.Context, agency domain.Agency) (domain.Agency, error)
	Update(ctx context.Context, update domain.AgencyUpdate) (domain.Agency, error)
	SoftDelete(ctx context.Context, id, usuarioModificacion string) error
	List(ctx context.Context, search string) ([]domain.Agency, error)
	// FindByID returns the active agency with that id. An agency given de
	// baja is reported as domain.ErrAgencyNotFound, same as a missing one.
	FindByID(ctx context.Context, id string) (domain.Agency, error)
}
