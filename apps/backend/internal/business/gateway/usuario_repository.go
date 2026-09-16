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
	// ListByRoles returns the users holding any of roles, ordered by
	// apellido and nombre. Users given de baja are only included when
	// includeInactive.
	ListByRoles(ctx context.Context, roles []domain.Rol, includeInactive bool) ([]domain.Usuario, error)
	Update(ctx context.Context, update domain.UsuarioUpdate) (domain.Usuario, error)
	SoftDelete(ctx context.Context, id, usuarioModificacion string) error
	// Reactivate clears fecha_baja on an inactive user. Returns
	// domain.ErrUsuarioNoEncontrado if id doesn't match an inactive user.
	Reactivate(ctx context.Context, id, usuarioModificacion string) error
}
