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
// aren't editable here — see domain.UsuarioUpdate. AgencyID only fills in
// the agency of a rol inmobiliaria user that has none.
type UpdateUserInput struct {
	ActorRoles []string
	Subject    string
	ID         string
	Nombre     *string
	Apellido   *string
	AgencyID   *string
}

// UpdateUser modifies an active user managed by this ABM (administrativo,
// escribano, inmobiliaria, agrimensor). Only callers with the
// administrador role may do this.
type UpdateUser interface {
	Execute(ctx context.Context, input UpdateUserInput) (domain.Usuario, error)
}

type updateUserUseCase struct {
	repository gateway.UserRepository
	agencies   gateway.AgencyRepository
}

func NewUpdateUser(repository gateway.UserRepository, agencies gateway.AgencyRepository) UpdateUser {
	return &updateUserUseCase{repository: repository, agencies: agencies}
}

func (useCase *updateUserUseCase) Execute(ctx context.Context, input UpdateUserInput) (domain.Usuario, error) {
	if !domain.HasRole(input.ActorRoles, domain.RolAdministrador) {
		return domain.Usuario{}, domain.ErrNoAutorizado
	}

	nombre := trimIfPresent(input.Nombre)
	apellido := trimIfPresent(input.Apellido)
	agencyID := trimIfPresent(input.AgencyID)
	if isBlank(nombre) || isBlank(apellido) {
		return domain.Usuario{}, domain.ErrPerfilInvalido
	}
	if isBlank(agencyID) {
		return domain.Usuario{}, domain.ErrAgenciaRequerida
	}
	if nombre == nil && apellido == nil && agencyID == nil {
		return domain.Usuario{}, domain.ErrUsuarioSinCambios
	}

	// A target of a role this ABM doesn't manage (administrador) is
	// reported as not found, so these routes can't be used to rename
	// accounts outside their scope.
	target, err := useCase.repository.FindByID(ctx, input.ID)
	if err != nil {
		return domain.Usuario{}, fromRepository(err)
	}
	if !esRolGestionable(target.Rol) {
		return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
	}

	if agencyID != nil {
		if err := useCase.checkAgencyAssignable(ctx, target, *agencyID); err != nil {
			return domain.Usuario{}, err
		}
	}

	actorID, err := resolveActorID(ctx, useCase.repository, input.Subject)
	if err != nil {
		return domain.Usuario{}, err
	}

	updated, err := useCase.repository.Update(ctx, domain.UsuarioUpdate{
		ID:                  input.ID,
		Nombre:              nombre,
		Apellido:            apellido,
		AgencyID:            agencyID,
		UsuarioModificacion: actorID,
	})
	if err != nil {
		return domain.Usuario{}, fromRepository(err)
	}

	return updated, nil
}

func (useCase *updateUserUseCase) checkAgencyAssignable(ctx context.Context, target domain.Usuario, agencyID string) error {
	if target.Rol != domain.RolInmobiliaria {
		return domain.ErrAgenciaNoAplica
	}
	if target.AgencyID != nil {
		return domain.ErrAgenciaYaAsignada
	}
	if _, err := useCase.agencies.FindByID(ctx, agencyID); err != nil {
		return fromRepository(err)
	}
	return nil
}
