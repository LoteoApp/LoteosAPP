package clients

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// UpdateClient modifies an existing active cliente. Only callers with the
// administrador, administrativo or inmobiliaria role may do this.
//
// This is a partial update: nombre, apellido, dni, celular and email are
// all optional. A nil field is left unchanged on the stored cliente — the
// caller only needs to send the fields it actually wants to change, as the
// PATCH /api/v1/clientes/{id} route implies. A field that is present but
// blank (after trimming) is rejected for nombre, apellido and dni, since a
// cliente can't be left without those.
type UpdateClient interface {
	Execute(
		ctx context.Context,
		actorRoles []string,
		subject, id string,
		nombre, apellido, dni, celular, email *string,
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
	subject, id string,
	nombre, apellido, dni, celular, email *string,
) (domain.Cliente, error) {
	if !hasRole(actorRoles, domain.RolAdministrador, domain.RolAdministrativo, domain.RolInmobiliaria) {
		return domain.Cliente{}, domain.ErrNoAutorizado
	}

	nombre = trimIfPresent(nombre)
	apellido = trimIfPresent(apellido)
	dni = trimIfPresent(dni)
	if isBlank(nombre) || isBlank(apellido) || isBlank(dni) {
		return domain.Cliente{}, domain.ErrClienteInvalido
	}

	actor, err := useCase.users.FindByAuthProviderID(ctx, subject)
	if err != nil {
		return domain.Cliente{}, wrapGatewayError(err)
	}

	updated, err := useCase.repository.Update(ctx, domain.ClienteUpdate{
		ID:                  id,
		Nombre:              nombre,
		Apellido:            apellido,
		DNI:                 dni,
		Celular:             celular,
		Email:               email,
		UsuarioModificacion: actor.ID,
	})
	if err != nil {
		return domain.Cliente{}, wrapGatewayError(err)
	}

	return updated, nil
}

// trimIfPresent trims s in place when it's present (non-nil), leaving an
// absent field (nil, meaning "unchanged") as nil.
func trimIfPresent(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	return &trimmed
}

// isBlank reports whether s is present but empty — i.e. the caller sent the
// field but it has no content, as opposed to not sending it at all.
func isBlank(s *string) bool {
	return s != nil && *s == ""
}
