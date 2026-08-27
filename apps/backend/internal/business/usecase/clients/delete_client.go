package clients

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// DeleteClient gives a cliente de baja (soft delete). Only callers with the
// administrador role may do this.
type DeleteClient interface {
	Execute(ctx context.Context, actorRoles []string, subject, id string) error
}

type deleteClientUseCase struct {
	repository gateway.ClienteRepository
	users      gateway.UserRepository
}

func NewDeleteClient(repository gateway.ClienteRepository, users gateway.UserRepository) DeleteClient {
	return &deleteClientUseCase{repository: repository, users: users}
}

func (useCase *deleteClientUseCase) Execute(ctx context.Context, actorRoles []string, subject, id string) error {
	if !hasRole(actorRoles, domain.RolAdministrador) {
		return domain.ErrNoAutorizado
	}

	actor, err := useCase.users.FindByAuthProviderID(ctx, subject)
	if err != nil {
		return wrapGatewayError(err)
	}

	return wrapGatewayError(useCase.repository.SoftDelete(ctx, id, actor.ID))
}
