package users

import (
	"context"
	"log/slog"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type CreateUserInput struct {
	ActorRoles []string
	Nombre     string
	Apellido   string
	Email      string
	Rol        string
	// InmobiliariaID is the agency a rol inmobiliaria user belongs to. It's
	// required for that role and must be empty for every other one.
	InmobiliariaID string
}

// CreateUser gives a new user access to the system. Only callers with the
// administrador role may do this. The role must be one of
// gestionableRoles — this ABM doesn't create other administrador accounts.
type CreateUser interface {
	Execute(ctx context.Context, input CreateUserInput) (domain.Usuario, string, error)
}

type createUserUseCase struct {
	repository gateway.UserRepository
	agencies   gateway.AgencyRepository
	identity   gateway.IdentityProvider
}

func NewCreateUser(repository gateway.UserRepository, agencies gateway.AgencyRepository, identity gateway.IdentityProvider) CreateUser {
	return &createUserUseCase{repository: repository, agencies: agencies, identity: identity}
}

// Execute creates the account in the identity provider first and, if
// persisting the local profile then fails, removes it again so no orphaned
// account is left behind.
func (useCase *createUserUseCase) Execute(ctx context.Context, input CreateUserInput) (domain.Usuario, string, error) {
	if !domain.HasRole(input.ActorRoles, domain.RolAdministrador) {
		return domain.Usuario{}, "", domain.ErrNoAutorizado
	}

	nombre := strings.TrimSpace(input.Nombre)
	apellido := strings.TrimSpace(input.Apellido)
	if nombre == "" || apellido == "" {
		return domain.Usuario{}, "", domain.ErrPerfilInvalido
	}

	email := strings.TrimSpace(input.Email)
	if !domain.EmailValido(email) {
		return domain.Usuario{}, "", domain.ErrEmailInvalido
	}

	rol := domain.Rol(input.Rol)
	if !esRolGestionable(rol) {
		return domain.Usuario{}, "", domain.ErrRolInvalido
	}

	inmobiliariaID, err := useCase.resolveAgency(ctx, rol, strings.TrimSpace(input.InmobiliariaID))
	if err != nil {
		return domain.Usuario{}, "", err
	}

	authProviderID, temporaryPassword, err := useCase.identity.CreateUser(ctx, email, input.Rol)
	if err != nil {
		return domain.Usuario{}, "", fromRepository(err)
	}

	usuario, err := useCase.repository.Create(ctx, domain.Usuario{
		AuthProviderID: authProviderID,
		Email:          email,
		Nombre:         nombre,
		Apellido:       apellido,
		Rol:            rol,
		InmobiliariaID: inmobiliariaID,
		PerfilCompleto: true,
	})
	if err != nil {
		if deleteErr := useCase.identity.DeleteUser(ctx, authProviderID); deleteErr != nil {
			slog.ErrorContext(ctx, "compensating identity provider delete failed after local persistence error",
				"auth_provider_id", authProviderID, "error", deleteErr)
		}
		return domain.Usuario{}, "", fromRepository(err)
	}

	return usuario, temporaryPassword, nil
}

// resolveAgency checks the agency against the role before the identity
// provider is touched, so a bad agency never leaves an account to compensate.
func (useCase *createUserUseCase) resolveAgency(ctx context.Context, rol domain.Rol, inmobiliariaID string) (*string, error) {
	if rol != domain.RolInmobiliaria {
		if inmobiliariaID != "" {
			return nil, domain.ErrInmobiliariaNoAplica
		}
		return nil, nil
	}

	if inmobiliariaID == "" {
		return nil, domain.ErrInmobiliariaRequerida
	}
	if _, err := useCase.agencies.FindByID(ctx, inmobiliariaID); err != nil {
		return nil, fromRepository(err)
	}

	return &inmobiliariaID, nil
}
