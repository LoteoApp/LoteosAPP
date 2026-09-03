package agencies

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// CreateAgencyInput carries the fields of a new agency.
type CreateAgencyInput struct {
	ActorRoles   []string
	Subject      string
	BusinessName string
	CUIT         *string
	Phone        *string
	Email        *string
}

// CreateAgency registers a new agency. Only callers with the
// administrador role may do this.
type CreateAgency interface {
	Execute(ctx context.Context, input CreateAgencyInput) (domain.Agency, error)
}

type createAgencyUseCase struct {
	repository gateway.AgencyRepository
	users      gateway.UserRepository
}

func NewCreateAgency(repository gateway.AgencyRepository, users gateway.UserRepository) CreateAgency {
	return &createAgencyUseCase{repository: repository, users: users}
}

func (useCase *createAgencyUseCase) Execute(ctx context.Context, input CreateAgencyInput) (domain.Agency, error) {
	if !hasRole(input.ActorRoles, domain.RolAdministrador) {
		return domain.Agency{}, domain.ErrNoAutorizado
	}

	razonSocial := strings.TrimSpace(input.BusinessName)
	if razonSocial == "" {
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

	actorID, err := resolveActorID(ctx, useCase.users, input.Subject)
	if err != nil {
		return domain.Agency{}, err
	}

	created, err := useCase.repository.Create(ctx, domain.Agency{
		BusinessName: razonSocial,
		CUIT:         cuit,
		Phone:        telefono,
		Email:        email,
		ModifiedBy:   actorID,
	})
	if err != nil {
		return domain.Agency{}, fromRepository(err)
	}

	return created, nil
}
