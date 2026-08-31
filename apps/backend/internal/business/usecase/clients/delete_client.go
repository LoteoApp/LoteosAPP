package clients

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type DeleteClientInput struct {
	ActorRoles []string
	Subject    string
	ID         string
}

// DeleteClient gives a cliente de baja (soft delete). Only callers with the
// administrador role may do this.
type DeleteClient interface {
	Execute(ctx context.Context, input DeleteClientInput) error
}

type deleteClientUseCase struct {
	repository gateway.ClienteRepository
	users      gateway.UserRepository
}

func NewDeleteClient(repository gateway.ClienteRepository, users gateway.UserRepository) DeleteClient {
	return &deleteClientUseCase{repository: repository, users: users}
}

func (useCase *deleteClientUseCase) Execute(ctx context.Context, input DeleteClientInput) error {
	if !hasRole(input.ActorRoles, domain.RolAdministrador) {
		return domain.ErrNoAutorizado
	}

	actorID, err := resolveActorID(ctx, useCase.users, input.Subject)
	if err != nil {
		return err
	}

	return useCase.repository.SoftDelete(ctx, input.ID, actorID)
}
