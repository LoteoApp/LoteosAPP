package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

type InmobiliariaRepository interface {
	Create(ctx context.Context, inmobiliaria domain.Inmobiliaria) (domain.Inmobiliaria, error)
	Update(ctx context.Context, update domain.InmobiliariaUpdate) (domain.Inmobiliaria, error)
	SoftDelete(ctx context.Context, id, usuarioModificacion string) error
	List(ctx context.Context, search string) ([]domain.Inmobiliaria, error)
}
