package users

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ListUsers lists the users this ABM manages (administrativo, escribano,
// inmobiliaria, agrimensor). Only callers with the administrador role may
// do this.
type ListUsers interface {
	Execute(ctx context.Context, actorRoles []string, includeInactive bool) ([]domain.Usuario, error)
}

type listUsersUseCase struct {
	repository gateway.UserRepository
}

func NewListUsers(repository gateway.UserRepository) ListUsers {
	return &listUsersUseCase{repository: repository}
}

func (useCase *listUsersUseCase) Execute(
	ctx context.Context,
	actorRoles []string,
	includeInactive bool,
) ([]domain.Usuario, error) {
	if !domain.HasRole(actorRoles, domain.RolAdministrador) {
		return nil, domain.ErrNoAutorizado
	}

	usuarios, err := useCase.repository.ListByRoles(ctx, gestionableRoles, includeInactive)
	if err != nil {
		return nil, fromRepository(err)
	}

	return usuarios, nil
}
