package agencies

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// UpdateAgencyInput carries a partial change to an existing inmobiliaria:
// every field is optional and a nil one is left unchanged.
type UpdateAgencyInput struct {
	ActorRoles  []string
	Subject     string
	ID          string
	RazonSocial *string
	CUIT        *string
	Telefono    *string
	Email       *string
}

// UpdateAgency modifies an existing active inmobiliaria. Only callers with
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
	Execute(ctx context.Context, input UpdateAgencyInput) (domain.Inmobiliaria, error)
}

type updateAgencyUseCase struct {
	repository gateway.InmobiliariaRepository
	users      gateway.UserRepository
}

func NewUpdateAgency(repository gateway.InmobiliariaRepository, users gateway.UserRepository) UpdateAgency {
	return &updateAgencyUseCase{repository: repository, users: users}
}

func (useCase *updateAgencyUseCase) Execute(ctx context.Context, input UpdateAgencyInput) (domain.Inmobiliaria, error) {
	if !hasRole(input.ActorRoles, domain.RolAdministrador) {
		return domain.Inmobiliaria{}, domain.ErrNoAutorizado
	}

	razonSocial := trimIfPresent(input.RazonSocial)
	if isBlank(razonSocial) {
		return domain.Inmobiliaria{}, domain.ErrInmobiliariaInvalida
	}

	cuit, err := normalizeOptionalCUIT(trimOptional(input.CUIT))
	if err != nil {
		return domain.Inmobiliaria{}, err
	}

	telefono := trimOptional(input.Telefono)
	email := trimOptional(input.Email)
	if err := validateOptionalEmail(email); err != nil {
		return domain.Inmobiliaria{}, err
	}

	if razonSocial == nil && cuit == nil && telefono == nil && email == nil {
		return domain.Inmobiliaria{}, domain.ErrInmobiliariaSinCambios
	}

	actorID, err := resolveActorID(ctx, useCase.users, input.Subject)
	if err != nil {
		return domain.Inmobiliaria{}, err
	}

	updated, err := useCase.repository.Update(ctx, domain.InmobiliariaUpdate{
		ID:                  input.ID,
		RazonSocial:         razonSocial,
		CUIT:                cuit,
		Telefono:            telefono,
		Email:               email,
		UsuarioModificacion: actorID,
	})
	if err != nil {
		return domain.Inmobiliaria{}, fromRepository(err)
	}

	return updated, nil
}
