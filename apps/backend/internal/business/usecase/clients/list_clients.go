package clients

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ListClients searches active clientes by nombre, apellido or dni. Every
// cliente carries DNI, celular and email, so listing is restricted to
// callers with the administrador, administrativo or inmobiliaria role —
// the same roles that may create or modify a cliente. agrimensor and
// escribano callers don't get a general client directory this way.
type ListClients interface {
	Execute(ctx context.Context, actorRoles []string, search string) ([]domain.Cliente, error)
}

type listClientsUseCase struct {
	repository gateway.ClienteRepository
}

func NewListClients(repository gateway.ClienteRepository) ListClients {
	return &listClientsUseCase{repository: repository}
}

func (useCase *listClientsUseCase) Execute(ctx context.Context, actorRoles []string, search string) ([]domain.Cliente, error) {
	if !hasRole(actorRoles, domain.RolAdministrador, domain.RolAdministrativo, domain.RolInmobiliaria) {
		return nil, domain.ErrNoAutorizado
	}

	clientes, err := useCase.repository.List(ctx, strings.TrimSpace(search))
	if err != nil {
		return nil, wrapGatewayError(err)
	}

	return clientes, nil
}
