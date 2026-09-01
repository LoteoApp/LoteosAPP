package agencies

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// CreateAgencyInput carries the fields of a new inmobiliaria.
type CreateAgencyInput struct {
	ActorRoles  []string
	Subject     string
	RazonSocial string
	CUIT        *string
	Telefono    *string
	Email       *string
}

// CreateAgency registers a new inmobiliaria. Only callers with the
// administrador role may do this.
type CreateAgency interface {
	Execute(ctx context.Context, input CreateAgencyInput) (domain.Inmobiliaria, error)
}

type createAgencyUseCase struct {
	repository gateway.InmobiliariaRepository
	users      gateway.UserRepository
}

func NewCreateAgency(repository gateway.InmobiliariaRepository, users gateway.UserRepository) CreateAgency {
	return &createAgencyUseCase{repository: repository, users: users}
}

func (useCase *createAgencyUseCase) Execute(ctx context.Context, input CreateAgencyInput) (domain.Inmobiliaria, error) {
	if !hasRole(input.ActorRoles, domain.RolAdministrador) {
		return domain.Inmobiliaria{}, domain.ErrNoAutorizado
	}

	razonSocial := strings.TrimSpace(input.RazonSocial)
	if razonSocial == "" {
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

	actorID, err := resolveActorID(ctx, useCase.users, input.Subject)
	if err != nil {
		return domain.Inmobiliaria{}, err
	}

	created, err := useCase.repository.Create(ctx, domain.Inmobiliaria{
		RazonSocial:         razonSocial,
		CUIT:                cuit,
		Telefono:            telefono,
		Email:               email,
		UsuarioModificacion: actorID,
	})
	if err != nil {
		return domain.Inmobiliaria{}, err
	}

	return created, nil
}
