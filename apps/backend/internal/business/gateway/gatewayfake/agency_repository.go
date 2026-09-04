package gatewayfake

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

// AgencyRepository is a fake gateway.AgencyRepository for tests.
type AgencyRepository struct {
	CreateCalls int
	CreateErr   error
	Created     domain.Agency

	UpdateCalls int
	UpdateErr   error
	Updated     domain.Agency
	UpdateInput domain.AgencyUpdate

	SoftDeleteCalls  int
	SoftDeleteErr    error
	SoftDeletedID    string
	SoftDeletedActor string

	ListCalls  int
	ListErr    error
	ListResult []domain.Agency
	ListSearch string
}

func (fake *AgencyRepository) Create(_ context.Context, agency domain.Agency) (domain.Agency, error) {
	fake.CreateCalls++
	if fake.CreateErr != nil {
		return domain.Agency{}, fake.CreateErr
	}
	if fake.Created.ID == "" {
		return agency, nil
	}
	return fake.Created, nil
}

func (fake *AgencyRepository) Update(_ context.Context, update domain.AgencyUpdate) (domain.Agency, error) {
	fake.UpdateCalls++
	fake.UpdateInput = update
	if fake.UpdateErr != nil {
		return domain.Agency{}, fake.UpdateErr
	}
	if fake.Updated.ID == "" {
		agency := domain.Agency{ID: update.ID, ModifiedBy: update.ModifiedBy}
		if update.BusinessName != nil {
			agency.BusinessName = *update.BusinessName
		}
		agency.CUIT = update.CUIT
		agency.Phone = update.Phone
		agency.Email = update.Email
		return agency, nil
	}
	return fake.Updated, nil
}

func (fake *AgencyRepository) SoftDelete(_ context.Context, id, usuarioModificacion string) error {
	fake.SoftDeleteCalls++
	fake.SoftDeletedID = id
	fake.SoftDeletedActor = usuarioModificacion
	return fake.SoftDeleteErr
}

func (fake *AgencyRepository) List(_ context.Context, search string) ([]domain.Agency, error) {
	fake.ListCalls++
	fake.ListSearch = search
	if fake.ListErr != nil {
		return nil, fake.ListErr
	}
	return fake.ListResult, nil
}
