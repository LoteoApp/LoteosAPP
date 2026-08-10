package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

type UserRepository interface {
	Create(ctx context.Context, usuario domain.Usuario) (domain.Usuario, error)
	FindByKeycloakID(ctx context.Context, keycloakID string) (domain.Usuario, error)
	UpdateProfile(ctx context.Context, keycloakID, nombre, apellido string) (domain.Usuario, error)
}
