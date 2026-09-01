package agencies

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type DeleteAgencyInput struct {
	ActorRoles []string
	Subject    string
	ID         string
}

// DeleteAgency gives an inmobiliaria de baja (soft delete). Only callers
// with the administrador role may do this.
type DeleteAgency interface {
	Execute(ctx context.Context, input DeleteAgencyInput) error
}

type deleteAgencyUseCase struct {
	repository gateway.InmobiliariaRepository
	users      gateway.UserRepository
}

func NewDeleteAgency(repository gateway.InmobiliariaRepository, users gateway.UserRepository) DeleteAgency {
	return &deleteAgencyUseCase{repository: repository, users: users}
}

func (useCase *deleteAgencyUseCase) Execute(ctx context.Context, input DeleteAgencyInput) error {
	if !hasRole(input.ActorRoles, domain.RolAdministrador) {
		return domain.ErrNoAutorizado
	}

	actorID, err := resolveActorID(ctx, useCase.users, input.Subject)
	if err != nil {
		return err
	}

	return useCase.repository.SoftDelete(ctx, input.ID, actorID)
}
