package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

// ClientRepository persiste clientes. usuarioModificacion es el id interno
// del usuario que ejecuta la operación, no el del proveedor de identidad.
type ClientRepository interface {
	Create(ctx context.Context, cliente domain.Cliente, usuarioModificacion string) (domain.Cliente, error)
	List(ctx context.Context, buscar string) ([]domain.Cliente, error)
	Update(ctx context.Context, cliente domain.Cliente, usuarioModificacion string) (domain.Cliente, error)
	SoftDelete(ctx context.Context, id, usuarioModificacion string) error
}
