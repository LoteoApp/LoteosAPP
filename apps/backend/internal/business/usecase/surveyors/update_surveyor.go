package surveyors

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// UpdateSurveyor modifies an active agrimensor. Only callers with the
// administrador role may do this.
//
// It is a partial update: nombre and apellido are both optional and a nil
// field is left unchanged, as PATCH /api/v1/agrimensores/{id} implies. A
// field that is present but blank is rejected, since an agrimensor can't be
// left without a name. Email is not editable here: it identifies the account
// in the identity provider, and changing it there is a separate operation.
type UpdateSurveyor interface {
	Execute(ctx context.Context, actorRoles []string, subject, id string, nombre, apellido *string) (domain.Usuario, error)
}

type updateSurveyorUseCase struct {
	repository gateway.UserRepository
}

func NewUpdateSurveyor(repository gateway.UserRepository) UpdateSurveyor {
	return &updateSurveyorUseCase{repository: repository}
}

// Execute reports a target of another rol as a missing agrimensor, so this
// route can't be used to rename, say, an administrador.
func (useCase *updateSurveyorUseCase) Execute(
	ctx context.Context,
	actorRoles []string,
	subject, id string,
	nombre, apellido *string,
) (domain.Usuario, error) {
	if !domain.HasRole(actorRoles, domain.RolAdministrador) {
		return domain.Usuario{}, domain.ErrNoAutorizado
	}

	nombre = trimIfPresent(nombre)
	apellido = trimIfPresent(apellido)
	if isBlank(nombre) || isBlank(apellido) {
		return domain.Usuario{}, domain.ErrPerfilInvalido
	}

	agrimensor, err := useCase.repository.FindByID(ctx, id)
	if err != nil {
		return domain.Usuario{}, asSurveyorNotFound(err)
	}
	if !agrimensor.EsAgrimensor() || !agrimensor.Activo() {
		return domain.Usuario{}, domain.ErrAgrimensorNoEncontrado
	}

	actor, err := useCase.repository.FindByAuthProviderID(ctx, subject)
	if err != nil {
		return domain.Usuario{}, fromRepository(err)
	}

	updated, err := useCase.repository.Update(ctx, domain.UsuarioUpdate{
		ID:                  id,
		Nombre:              nombre,
		Apellido:            apellido,
		UsuarioModificacion: actor.ID,
	})
	if err != nil {
		return domain.Usuario{}, fromRepository(err)
	}

	return updated, nil
}

func trimIfPresent(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func isBlank(value *string) bool {
	return value != nil && *value == ""
}
