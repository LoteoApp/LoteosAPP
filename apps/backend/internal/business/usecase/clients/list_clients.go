package clients

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type ListClientsInput struct {
	ActorRoles []string
	Search     string
}

// ListClients searches active clientes by nombre, apellido or dni. Every
// cliente carries DNI, celular and email, so listing is restricted to
// callers with the administrador, administrativo or inmobiliaria role —
// the same roles that may create or modify a cliente. agrimensor and
// escribano callers don't get a general client directory this way.
type ListClients interface {
	Execute(ctx context.Context, input ListClientsInput) ([]domain.Cliente, error)
}

type listClientsUseCase struct {
	repository gateway.ClienteRepository
}

func NewListClients(repository gateway.ClienteRepository) ListClients {
	return &listClientsUseCase{repository: repository}
}

func (useCase *listClientsUseCase) Execute(ctx context.Context, input ListClientsInput) ([]domain.Cliente, error) {
	if !hasRole(input.ActorRoles, domain.RolAdministrador, domain.RolAdministrativo, domain.RolInmobiliaria) {
		return nil, domain.ErrNoAutorizado
	}

	clientes, err := useCase.repository.List(ctx, strings.TrimSpace(input.Search))
	if err != nil {
		return nil, err
	}

	return clientes, nil
}
