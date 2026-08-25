package clients

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// UpdateClient modifies an existing active cliente. Only callers with the
// administrador, administrativo or inmobiliaria role may do this.
type UpdateClient interface {
	Execute(
		ctx context.Context,
		actorRoles []string,
		subject, id, nombre, apellido, dni string,
		celular, email *string,
	) (domain.Cliente, error)
}

type updateClientUseCase struct {
	repository gateway.ClienteRepository
	users      gateway.UserRepository
}

func NewUpdateClient(repository gateway.ClienteRepository, users gateway.UserRepository) UpdateClient {
	return &updateClientUseCase{repository: repository, users: users}
}

func (useCase *updateClientUseCase) Execute(
	ctx context.Context,
	actorRoles []string,
	subject, id, nombre, apellido, dni string,
	celular, email *string,
) (domain.Cliente, error) {
	if !hasRole(actorRoles, domain.RolAdministrador, domain.RolAdministrativo, domain.RolInmobiliaria) {
		return domain.Cliente{}, domain.ErrNoAutorizado
	}

	nombre = strings.TrimSpace(nombre)
	apellido = strings.TrimSpace(apellido)
	dni = strings.TrimSpace(dni)
	if nombre == "" || apellido == "" || dni == "" {
		return domain.Cliente{}, domain.ErrClienteInvalido
	}

	actor, err := useCase.users.FindByAuthProviderID(ctx, subject)
	if err != nil {
		return domain.Cliente{}, err
	}

	return useCase.repository.Update(ctx, domain.Cliente{
		ID:                  id,
		Nombre:              nombre,
		Apellido:            apellido,
		DNI:                 dni,
		Celular:             celular,
		Email:               email,
		UsuarioModificacion: actor.ID,
	})
}
