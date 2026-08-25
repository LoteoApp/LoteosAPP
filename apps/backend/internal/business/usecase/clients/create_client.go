package clients

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// CreateClient registers a new cliente. Only callers with the
// administrador, administrativo or inmobiliaria role may do this.
type CreateClient interface {
	Execute(
		ctx context.Context,
		actorRoles []string,
		subject, nombre, apellido, dni string,
		celular, email *string,
	) (domain.Cliente, error)
}

type createClientUseCase struct {
	repository gateway.ClienteRepository
	users      gateway.UserRepository
}

func NewCreateClient(repository gateway.ClienteRepository, users gateway.UserRepository) CreateClient {
	return &createClientUseCase{repository: repository, users: users}
}

func (useCase *createClientUseCase) Execute(
	ctx context.Context,
	actorRoles []string,
	subject, nombre, apellido, dni string,
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

	return useCase.repository.Create(ctx, domain.Cliente{
		Nombre:              nombre,
		Apellido:            apellido,
		DNI:                 dni,
		Celular:             celular,
		Email:               email,
		UsuarioModificacion: actor.ID,
	})
}

// hasRole reports whether actorRoles contains any of allowed.
func hasRole(actorRoles []string, allowed ...string) bool {
	for _, role := range actorRoles {
		for _, want := range allowed {
			if role == want {
				return true
			}
		}
	}
	return false
}
