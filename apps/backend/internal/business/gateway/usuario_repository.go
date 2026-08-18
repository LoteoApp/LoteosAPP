package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

type UserRepository interface {
	Create(ctx context.Context, usuario domain.Usuario) (domain.Usuario, error)
	FindByAuthProviderID(ctx context.Context, authProviderID string) (domain.Usuario, error)
	UpdateProfile(ctx context.Context, authProviderID, nombre, apellido string) (domain.Usuario, error)
}
