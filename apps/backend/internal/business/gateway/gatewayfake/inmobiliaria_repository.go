package gatewayfake

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

// InmobiliariaRepository is a fake gateway.InmobiliariaRepository for tests.
type InmobiliariaRepository struct {
	CreateCalls int
	CreateErr   error
	Created     domain.Inmobiliaria

	UpdateCalls int
	UpdateErr   error
	Updated     domain.Inmobiliaria
	UpdateInput domain.InmobiliariaUpdate

	SoftDeleteCalls  int
	SoftDeleteErr    error
	SoftDeletedID    string
	SoftDeletedActor string

	ListCalls  int
	ListErr    error
	ListResult []domain.Inmobiliaria
	ListSearch string
}

func (fake *InmobiliariaRepository) Create(_ context.Context, inmobiliaria domain.Inmobiliaria) (domain.Inmobiliaria, error) {
	fake.CreateCalls++
	if fake.CreateErr != nil {
		return domain.Inmobiliaria{}, fake.CreateErr
	}
	if fake.Created.ID == "" {
		return inmobiliaria, nil
	}
	return fake.Created, nil
}

func (fake *InmobiliariaRepository) Update(_ context.Context, update domain.InmobiliariaUpdate) (domain.Inmobiliaria, error) {
	fake.UpdateCalls++
	fake.UpdateInput = update
	if fake.UpdateErr != nil {
		return domain.Inmobiliaria{}, fake.UpdateErr
	}
	if fake.Updated.ID == "" {
		inmobiliaria := domain.Inmobiliaria{ID: update.ID, UsuarioModificacion: update.UsuarioModificacion}
		if update.RazonSocial != nil {
			inmobiliaria.RazonSocial = *update.RazonSocial
		}
		inmobiliaria.CUIT = update.CUIT
		inmobiliaria.Telefono = update.Telefono
		inmobiliaria.Email = update.Email
		return inmobiliaria, nil
	}
	return fake.Updated, nil
}

func (fake *InmobiliariaRepository) SoftDelete(_ context.Context, id, usuarioModificacion string) error {
	fake.SoftDeleteCalls++
	fake.SoftDeletedID = id
	fake.SoftDeletedActor = usuarioModificacion
	return fake.SoftDeleteErr
}

func (fake *InmobiliariaRepository) List(_ context.Context, search string) ([]domain.Inmobiliaria, error) {
	fake.ListCalls++
	fake.ListSearch = search
	if fake.ListErr != nil {
		return nil, fake.ListErr
	}
	return fake.ListResult, nil
}
