package surveyors

import (
	"context"
	"log/slog"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// CreateSurveyor gives a new agrimensor access to the system. Only callers
// with the administrador role may do this.
type CreateSurveyor interface {
	Execute(ctx context.Context, actorRoles []string, nombre, apellido, email string) (domain.Usuario, string, error)
}

type createSurveyorUseCase struct {
	repository gateway.UserRepository
	identity   gateway.IdentityProvider
}

func NewCreateSurveyor(repository gateway.UserRepository, identity gateway.IdentityProvider) CreateSurveyor {
	return &createSurveyorUseCase{repository: repository, identity: identity}
}

// Execute creates the account in the identity provider first and, if
// persisting the local profile then fails, removes it again so no orphaned
// account is left behind.
func (useCase *createSurveyorUseCase) Execute(
	ctx context.Context,
	actorRoles []string,
	nombre, apellido, email string,
) (domain.Usuario, string, error) {
	if !domain.HasRole(actorRoles, domain.RolAdministrador) {
		return domain.Usuario{}, "", domain.ErrNoAutorizado
	}

	nombre = strings.TrimSpace(nombre)
	apellido = strings.TrimSpace(apellido)
	if !domain.PerfilEstaCompleto(nombre, apellido) {
		return domain.Usuario{}, "", domain.ErrPerfilInvalido
	}

	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return domain.Usuario{}, "", domain.ErrEmailInvalido
	}

	authProviderID, temporaryPassword, err := useCase.identity.CreateUser(ctx, email, domain.RolAgrimensor)
	if err != nil {
		return domain.Usuario{}, "", fromRepository(err)
	}

	agrimensor, err := useCase.repository.Create(ctx, domain.Usuario{
		AuthProviderID: authProviderID,
		Email:          email,
		Nombre:         nombre,
		Apellido:       apellido,
		Rol:            domain.RolAgrimensor,
		PerfilCompleto: true,
	})
	if err != nil {
		if deleteErr := useCase.identity.DeleteUser(ctx, authProviderID); deleteErr != nil {
			slog.ErrorContext(ctx, "compensating identity provider delete failed after local persistence error",
				"auth_provider_id", authProviderID, "error", deleteErr)
		}
		return domain.Usuario{}, "", fromRepository(err)
	}

	return agrimensor, temporaryPassword, nil
}
