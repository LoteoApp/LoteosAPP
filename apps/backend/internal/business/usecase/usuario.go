package usecase

import (
	"context"
	"log/slog"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type UserService struct {
	repository gateway.UserRepository
	identity   gateway.IdentityProvider
}

func NewUserService(repository gateway.UserRepository, identity gateway.IdentityProvider) *UserService {
	return &UserService{repository: repository, identity: identity}
}

// CreateUser gives a new user access to the system. Only callers with the
// administrador role may do this. It creates the account in the identity
// provider first and, if persisting the local profile then fails, removes
// it again so no orphaned account is left behind.
func (service *UserService) CreateUser(
	ctx context.Context,
	actorRoles []string,
	email, rol string,
) (domain.Usuario, string, error) {
	if !hasRole(actorRoles, domain.RolAdministrador) {
		return domain.Usuario{}, "", domain.ErrNoAutorizado
	}

	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return domain.Usuario{}, "", domain.ErrEmailInvalido
	}

	if !domain.RolValido(rol) {
		return domain.Usuario{}, "", domain.ErrRolInvalido
	}

	authProviderID, temporaryPassword, err := service.identity.CreateUser(ctx, email, rol)
	if err != nil {
		return domain.Usuario{}, "", err
	}

	usuario, err := service.repository.Create(ctx, domain.Usuario{
		AuthProviderID: authProviderID,
		Email:          email,
		Rol:            rol,
	})
	if err != nil {
		if deleteErr := service.identity.DeleteUser(ctx, authProviderID); deleteErr != nil {
			slog.ErrorContext(ctx, "compensating identity provider delete failed after local persistence error",
				"auth_provider_id", authProviderID, "error", deleteErr)
		}
		return domain.Usuario{}, "", err
	}

	return usuario, temporaryPassword, nil
}

// CompleteProfile lets the authenticated user (identified by subject, the
// identity provider's user ID) fill in their own name.
func (service *UserService) CompleteProfile(ctx context.Context, subject, nombre, apellido string) (domain.Usuario, error) {
	nombre = strings.TrimSpace(nombre)
	apellido = strings.TrimSpace(apellido)
	if nombre == "" || apellido == "" {
		return domain.Usuario{}, domain.ErrPerfilInvalido
	}

	return service.repository.UpdateProfile(ctx, subject, nombre, apellido)
}

func hasRole(roles []string, role string) bool {
	for _, candidate := range roles {
		if candidate == role {
			return true
		}
	}
	return false
}
