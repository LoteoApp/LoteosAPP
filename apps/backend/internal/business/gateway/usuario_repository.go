package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

type UserRepository interface {
	Create(ctx context.Context, usuario domain.Usuario) (domain.Usuario, error)
	FindByAuthProviderID(ctx context.Context, authProviderID string) (domain.Usuario, error)
	FindByID(ctx context.Context, id string) (domain.Usuario, error)
	UpdateProfile(ctx context.Context, authProviderID, nombre, apellido string) (domain.Usuario, error)
	// ListByRol returns the users holding rol, ordered by apellido and
	// nombre. Users given de baja are only included when includeInactive.
	ListByRol(ctx context.Context, rol domain.Rol, includeInactive bool) ([]domain.Usuario, error)
	Update(ctx context.Context, update domain.UsuarioUpdate) (domain.Usuario, error)
	SoftDelete(ctx context.Context, id, usuarioModificacion string) error
}
