package users

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// DeactivateUser gives a user managed by this ABM (administrativo,
// escribano, inmobiliaria) de baja. The row is kept and only marked with
// its fecha de baja, so the records that name it stay readable. Only
// callers with the administrador role may do this.
type DeactivateUser interface {
	Execute(ctx context.Context, actorRoles []string, subject, id string) error
}

type deactivateUserUseCase struct {
	repository gateway.UserRepository
}

func NewDeactivateUser(repository gateway.UserRepository) DeactivateUser {
	return &deactivateUserUseCase{repository: repository}
}

func (useCase *deactivateUserUseCase) Execute(
	ctx context.Context,
	actorRoles []string,
	subject, id string,
) error {
	if !domain.HasRole(actorRoles, domain.RolAdministrador) {
		return domain.ErrNoAutorizado
	}

	target, err := useCase.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !esRolGestionable(target.Rol) {
		return domain.ErrUsuarioNoEncontrado
	}
	if !target.Activo() {
		return domain.ErrUsuarioDadoDeBaja
	}

	actorID, err := resolveActorID(ctx, useCase.repository, subject)
	if err != nil {
		return err
	}

	return useCase.repository.SoftDelete(ctx, id, actorID)
}
