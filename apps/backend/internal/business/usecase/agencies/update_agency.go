package agencies

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// UpdateAgencyInput carries a partial change to an existing inmobiliaria:
// every field is optional and a nil one is left unchanged.
type UpdateAgencyInput struct {
	ActorRoles   []string
	Subject      string
	ID           string
	BusinessName *string
	CUIT         *string
	Phone        *string
	Email        *string
}

// UpdateAgency modifies an existing active agency. Only callers with
// the administrador role may do this.
//
// This is a partial update, as the PATCH /api/v1/inmobiliarias/{id} route
// implies: a nil field is left unchanged. A razón social that is present but
// blank is rejected, since an inmobiliaria can't be left without one; a blank
// cuit, telefono or email is read as "not sent", because this type can't
// clear them to null. A request that carries no field at all is rejected
// instead of stamping usuario_modificacion and fecha_modificacion for a
// change that isn't one.
type UpdateAgency interface {
	Execute(ctx context.Context, input UpdateAgencyInput) (domain.Agency, error)
}

type updateAgencyUseCase struct {
	repository gateway.AgencyRepository
	users      gateway.UserRepository
}

func NewUpdateAgency(repository gateway.AgencyRepository, users gateway.UserRepository) UpdateAgency {
	return &updateAgencyUseCase{repository: repository, users: users}
}

func (useCase *updateAgencyUseCase) Execute(ctx context.Context, input UpdateAgencyInput) (domain.Agency, error) {
	if !hasRole(input.ActorRoles, domain.RolAdministrador) {
		return domain.Agency{}, domain.ErrNoAutorizado
	}

	razonSocial := trimIfPresent(input.BusinessName)
	if isBlank(razonSocial) {
		return domain.Agency{}, domain.ErrInvalidAgency
	}

	cuit, err := normalizeOptionalCUIT(trimOptional(input.CUIT))
	if err != nil {
		return domain.Agency{}, err
	}

	telefono := trimOptional(input.Phone)
	email := trimOptional(input.Email)
	if err := validateOptionalEmail(email); err != nil {
		return domain.Agency{}, err
	}

	if razonSocial == nil && cuit == nil && telefono == nil && email == nil {
		return domain.Agency{}, domain.ErrEmptyAgencyUpdate
	}

	actorID, err := resolveActorID(ctx, useCase.users, input.Subject)
	if err != nil {
		return domain.Agency{}, err
	}

	updated, err := useCase.repository.Update(ctx, domain.AgencyUpdate{
		ID:           input.ID,
		BusinessName: razonSocial,
		CUIT:         cuit,
		Phone:        telefono,
		Email:        email,
		ModifiedBy:   actorID,
	})
	if err != nil {
		return domain.Agency{}, fromRepository(err)
	}

	return updated, nil
}
