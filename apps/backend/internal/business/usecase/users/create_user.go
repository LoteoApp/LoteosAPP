package users

import (
	"context"
	"log/slog"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// CreateUser gives a new user access to the system. Only callers with the
// administrador role may do this.
type CreateUser interface {
	Execute(ctx context.Context, actorRoles []string, email, rol string) (domain.Usuario, string, error)
}

type createUserUseCase struct {
	repository gateway.UserRepository
	identity   gateway.IdentityProvider
}

func NewCreateUser(repository gateway.UserRepository, identity gateway.IdentityProvider) CreateUser {
	return &createUserUseCase{repository: repository, identity: identity}
}

// Execute creates the account in the identity provider first and, if
// persisting the local profile then fails, removes it again so no orphaned
// account is left behind.
func (useCase *createUserUseCase) Execute(
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

	if !domain.Rol(rol).Valido() {
		return domain.Usuario{}, "", domain.ErrRolInvalido
	}

	authProviderID, temporaryPassword, err := useCase.identity.CreateUser(ctx, email, rol)
	if err != nil {
		return domain.Usuario{}, "", err
	}

	usuario, err := useCase.repository.Create(ctx, domain.Usuario{
		AuthProviderID: authProviderID,
		Email:          email,
		Rol:            domain.Rol(rol),
	})
	if err != nil {
		if deleteErr := useCase.identity.DeleteUser(ctx, authProviderID); deleteErr != nil {
			slog.ErrorContext(ctx, "compensating identity provider delete failed after local persistence error",
				"auth_provider_id", authProviderID, "error", deleteErr)
		}
		return domain.Usuario{}, "", err
	}

	return usuario, temporaryPassword, nil
}

func hasRole(roles []string, role string) bool {
	for _, candidate := range roles {
		if candidate == role {
			return true
		}
	}
	return false
}
