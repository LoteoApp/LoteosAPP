package users

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// UpdateUserInput carries a partial change to an existing user: nombre and
// apellido are both optional and a nil field is left unchanged, as
// PATCH /api/v1/usuarios/{id} implies. A field that is present but blank is
// rejected, since a user can't be left without a name. Email and role
// aren't editable here — see domain.UsuarioUpdate.
type UpdateUserInput struct {
	ActorRoles []string
	Subject    string
	ID         string
	Nombre     *string
	Apellido   *string
}

// UpdateUser modifies an active user managed by this ABM (administrativo,
// escribano, inmobiliaria). Only callers with the administrador role may do
// this.
type UpdateUser interface {
	Execute(ctx context.Context, input UpdateUserInput) (domain.Usuario, error)
}

type updateUserUseCase struct {
	repository gateway.UserRepository
}

func NewUpdateUser(repository gateway.UserRepository) UpdateUser {
	return &updateUserUseCase{repository: repository}
}

func (useCase *updateUserUseCase) Execute(ctx context.Context, input UpdateUserInput) (domain.Usuario, error) {
	if !domain.HasRole(input.ActorRoles, domain.RolAdministrador) {
		return domain.Usuario{}, domain.ErrNoAutorizado
	}

	nombre := trimIfPresent(input.Nombre)
	apellido := trimIfPresent(input.Apellido)
	if isBlank(nombre) || isBlank(apellido) {
		return domain.Usuario{}, domain.ErrPerfilInvalido
	}
	if nombre == nil && apellido == nil {
		return domain.Usuario{}, domain.ErrUsuarioSinCambios
	}

	// A target of a role this ABM doesn't manage (agrimensor,
	// administrador) is reported as not found, so these routes can't be
	// used to rename accounts outside their scope.
	target, err := useCase.repository.FindByID(ctx, input.ID)
	if err != nil {
		return domain.Usuario{}, err
	}
	if !esRolGestionable(target.Rol) {
		return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
	}

	actorID, err := resolveActorID(ctx, useCase.repository, input.Subject)
	if err != nil {
		return domain.Usuario{}, err
	}

	updated, err := useCase.repository.Update(ctx, domain.UsuarioUpdate{
		ID:                  input.ID,
		Nombre:              nombre,
		Apellido:            apellido,
		UsuarioModificacion: actorID,
	})
	if err != nil {
		return domain.Usuario{}, err
	}

	return updated, nil
}
