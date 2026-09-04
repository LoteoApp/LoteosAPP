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
}
