package clients

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// UpdateClientInput carries a partial change to an existing cliente: every
// field is optional and a nil one is left unchanged.
type UpdateClientInput struct {
	ActorRoles []string
	Subject    string
	ID         string
	Nombre     *string
	Apellido   *string
	DNI        *string
	Celular    *string
	Email      *string
}

// UpdateClient modifies an existing active cliente. Only callers with the
// administrador, administrativo or inmobiliaria role may do this.
//
// This is a partial update: a nil field is left unchanged on the stored
// cliente, as the PATCH /api/v1/clientes/{id} route implies. A field that
// is present but blank (after trimming) is rejected for nombre, apellido
// and dni, since a cliente can't be left without those; a blank celular or
// email is read as "not sent", because this type can't clear them to null.
// A request that carries no field at all is rejected instead of stamping
// usuario_modificacion and fecha_modificacion for a change that isn't one.
type UpdateClient interface {
	Execute(ctx context.Context, input UpdateClientInput) (domain.Cliente, error)
}

type updateClientUseCase struct {
	repository gateway.ClienteRepository
	users      gateway.UserRepository
}

func NewUpdateClient(repository gateway.ClienteRepository, users gateway.UserRepository) UpdateClient {
	return &updateClientUseCase{repository: repository, users: users}
}

func (useCase *updateClientUseCase) Execute(ctx context.Context, input UpdateClientInput) (domain.Cliente, error) {
	if !hasRole(input.ActorRoles, domain.RolAdministrador, domain.RolAdministrativo, domain.RolInmobiliaria) {
		return domain.Cliente{}, domain.ErrNoAutorizado
	}

	nombre := trimIfPresent(input.Nombre)
	apellido := trimIfPresent(input.Apellido)
	dni := trimIfPresent(input.DNI)
	if isBlank(nombre) || isBlank(apellido) || isBlank(dni) {
		return domain.Cliente{}, domain.ErrClienteInvalido
	}

	celular := trimOptional(input.Celular)
	email := trimOptional(input.Email)
	if err := validateOptionalEmail(email); err != nil {
		return domain.Cliente{}, err
	}

	if nombre == nil && apellido == nil && dni == nil && celular == nil && email == nil {
		return domain.Cliente{}, domain.ErrClienteSinCambios
	}

	actorID, err := resolveActorID(ctx, useCase.users, input.Subject)
	if err != nil {
		return domain.Cliente{}, err
	}

	updated, err := useCase.repository.Update(ctx, domain.ClienteUpdate{
		ID:                  input.ID,
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

	return updated, nil
}
