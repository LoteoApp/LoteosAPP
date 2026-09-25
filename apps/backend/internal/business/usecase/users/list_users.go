package users

import (
	"context"
	"log/slog"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ListUsers lists the users this ABM manages (administrativo, escribano,
// inmobiliaria, agrimensor). Only callers with the administrador role may
// do this. Each user carries InvitacionAceptada when the identity provider
// could be asked.
type ListUsers interface {
	Execute(ctx context.Context, actorRoles []string, includeInactive bool) ([]domain.Usuario, error)
}

type listUsersUseCase struct {
	repository gateway.UserRepository
	identity   gateway.IdentityProvider
}

func NewListUsers(repository gateway.UserRepository, identity gateway.IdentityProvider) ListUsers {
	return &listUsersUseCase{repository: repository, identity: identity}
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

	confirmed, err := useCase.identity.ConfirmedAccountIDs(ctx)
	if err != nil {
		slog.WarnContext(ctx, "could not tell which invitations were accepted, listing without it", "error", err)
		return usuarios, nil
	}

	for index := range usuarios {
		accepted := confirmed[usuarios[index].AuthProviderID]
		usuarios[index].InvitacionAceptada = &accepted
	}

	return usuarios, nil
}
