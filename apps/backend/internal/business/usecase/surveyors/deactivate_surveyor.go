package surveyors

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// DeactivateSurveyor gives an agrimensor de baja. The row is kept and only
// marked with its fecha de baja, so the loteos it worked on and the records
// that name it stay readable. Only callers with the administrador role may do
// this.
type DeactivateSurveyor interface {
	Execute(ctx context.Context, actorRoles []string, subject, id string) error
}

type deactivateSurveyorUseCase struct {
	repository gateway.UserRepository
}

func NewDeactivateSurveyor(repository gateway.UserRepository) DeactivateSurveyor {
	return &deactivateSurveyorUseCase{repository: repository}
}

func (useCase *deactivateSurveyorUseCase) Execute(
	ctx context.Context,
	actorRoles []string,
	subject, id string,
) error {
	if !domain.HasRole(actorRoles, domain.RolAdministrador) {
		return domain.ErrNoAutorizado
	}

	agrimensor, err := useCase.repository.FindByID(ctx, id)
	if err != nil {
		return asSurveyorNotFound(err)
	}
	if !agrimensor.EsAgrimensor() {
		return domain.ErrAgrimensorNoEncontrado
	}
	if !agrimensor.Activo() {
		return domain.ErrAgrimensorDadoDeBaja
	}

	actorID, err := resolveActorID(ctx, useCase.repository, subject)
	if err != nil {
		return err
	}

	return fromRepository(useCase.repository.SoftDelete(ctx, id, actorID))
}
