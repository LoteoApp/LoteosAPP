package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

type ClienteRepository interface {
	Create(ctx context.Context, cliente domain.Cliente) (domain.Cliente, error)
	Update(ctx context.Context, cliente domain.Cliente) (domain.Cliente, error)
	SoftDelete(ctx context.Context, id, usuarioModificacion string) error
	List(ctx context.Context, search string) ([]domain.Cliente, error)
}
