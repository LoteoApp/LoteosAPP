package gatewayfake

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

// ClienteRepository is a fake gateway.ClienteRepository for tests.
type ClienteRepository struct {
	CreateCalls int
	CreateErr   error
	Created     domain.Cliente

	UpdateCalls int
	UpdateErr   error
	Updated     domain.Cliente
	UpdateInput domain.ClienteUpdate

	SoftDeleteCalls  int
	SoftDeleteErr    error
	SoftDeletedID    string
	SoftDeletedActor string

	ListCalls  int
	ListErr    error
	ListResult []domain.Cliente
	ListSearch string
}

func (fake *ClienteRepository) Create(_ context.Context, cliente domain.Cliente) (domain.Cliente, error) {
	fake.CreateCalls++
	if fake.CreateErr != nil {
		return domain.Cliente{}, fake.CreateErr
	}
	if fake.Created.ID == "" {
		return cliente, nil
	}
	return fake.Created, nil
}

func (fake *ClienteRepository) Update(_ context.Context, update domain.ClienteUpdate) (domain.Cliente, error) {
	fake.UpdateCalls++
	fake.UpdateInput = update
	if fake.UpdateErr != nil {
		return domain.Cliente{}, fake.UpdateErr
	}
	if fake.Updated.ID == "" {
		cliente := domain.Cliente{ID: update.ID, UsuarioModificacion: update.UsuarioModificacion}
		if update.Nombre != nil {
			cliente.Nombre = *update.Nombre
		}
		if update.Apellido != nil {
			cliente.Apellido = *update.Apellido
		}
		if update.DNI != nil {
			cliente.DNI = *update.DNI
		}
		cliente.Celular = update.Celular
		cliente.Email = update.Email
		return cliente, nil
	}
	return fake.Updated, nil
}

func (fake *ClienteRepository) SoftDelete(_ context.Context, id, usuarioModificacion string) error {
	fake.SoftDeleteCalls++
	fake.SoftDeletedID = id
	fake.SoftDeletedActor = usuarioModificacion
	return fake.SoftDeleteErr
}

func (fake *ClienteRepository) List(_ context.Context, search string) ([]domain.Cliente, error) {
	fake.ListCalls++
	fake.ListSearch = search
	if fake.ListErr != nil {
		return nil, fake.ListErr
	}
	return fake.ListResult, nil
}
