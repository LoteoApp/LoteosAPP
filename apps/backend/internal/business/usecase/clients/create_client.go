package clients

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// CreateClientInput carries the fields of a new cliente. It's a struct and
// not a list of parameters so nombre, apellido and dni can't be swapped at
// a call site without the compiler noticing.
type CreateClientInput struct {
	ActorRoles []string
	Subject    string
	Nombre     string
	Apellido   string
	DNI        string
	Celular    *string
	Email      *string
}

// CreateClient registers a new cliente. Only callers with the
// administrador, administrativo or inmobiliaria role may do this.
type CreateClient interface {
	Execute(ctx context.Context, input CreateClientInput) (domain.Cliente, error)
}

type createClientUseCase struct {
	repository gateway.ClienteRepository
	users      gateway.UserRepository
}

func NewCreateClient(repository gateway.ClienteRepository, users gateway.UserRepository) CreateClient {
	return &createClientUseCase{repository: repository, users: users}
}

func (useCase *createClientUseCase) Execute(ctx context.Context, input CreateClientInput) (domain.Cliente, error) {
	if !hasRole(input.ActorRoles, domain.RolAdministrador, domain.RolAdministrativo, domain.RolInmobiliaria) {
		return domain.Cliente{}, domain.ErrNoAutorizado
	}

	nombre := strings.TrimSpace(input.Nombre)
	apellido := strings.TrimSpace(input.Apellido)
	dni := strings.TrimSpace(input.DNI)
	if nombre == "" || apellido == "" || dni == "" {
		return domain.Cliente{}, domain.ErrClienteInvalido
	}

	celular := trimOptional(input.Celular)
	email := trimOptional(input.Email)
	if err := validateOptionalEmail(email); err != nil {
		return domain.Cliente{}, err
	}

	actorID, err := resolveActorID(ctx, useCase.users, input.Subject)
	if err != nil {
		return domain.Cliente{}, err
	}

	created, err := useCase.repository.Create(ctx, domain.Cliente{
		Nombre:              nombre,
		Apellido:            apellido,
		DNI:                 dni,
		Celular:             celular,
		Email:               email,
		UsuarioModificacion: actorID,
	})
	if err != nil {
		return domain.Cliente{}, err
	}

	return created, nil
}
