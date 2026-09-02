package users

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ReactivateUser undoes a baja on a user managed by this ABM (administrativo,
// escribano, inmobiliaria), the inverse of DeactivateUser. Only callers with
// the administrador role may do this.
type ReactivateUser interface {
	Execute(ctx context.Context, actorRoles []string, subject, id string) (domain.Usuario, error)
}

type reactivateUserUseCase struct {
	repository gateway.UserRepository
}

func NewReactivateUser(repository gateway.UserRepository) ReactivateUser {
	return &reactivateUserUseCase{repository: repository}
}

func (useCase *reactivateUserUseCase) Execute(
	ctx context.Context,
	actorRoles []string,
	subject, id string,
) (domain.Usuario, error) {
	if !domain.HasRole(actorRoles, domain.RolAdministrador) {
		return domain.Usuario{}, domain.ErrNoAutorizado
	}

	target, err := useCase.repository.FindByID(ctx, id)
	if err != nil {
		return domain.Usuario{}, err
	}
	if !esRolGestionable(target.Rol) {
		return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
	}
	if target.Activo() {
		return domain.Usuario{}, domain.ErrUsuarioYaActivo
	}

	actorID, err := resolveActorID(ctx, useCase.repository, subject)
	if err != nil {
		return domain.Usuario{}, err
	}

	if err := useCase.repository.Reactivate(ctx, id, actorID); err != nil {
		return domain.Usuario{}, err
	}

	target.FechaBaja = nil
	return target, nil
}
