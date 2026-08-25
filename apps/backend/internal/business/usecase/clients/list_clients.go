package clients

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ListClients searches active clientes by nombre, apellido or dni. Any
// authenticated caller may do this, regardless of role.
type ListClients interface {
	Execute(ctx context.Context, search string) ([]domain.Cliente, error)
}

type listClientsUseCase struct {
	repository gateway.ClienteRepository
}

func NewListClients(repository gateway.ClienteRepository) ListClients {
	return &listClientsUseCase{repository: repository}
}

func (useCase *listClientsUseCase) Execute(ctx context.Context, search string) ([]domain.Cliente, error) {
	return useCase.repository.List(ctx, strings.TrimSpace(search))
}
