package surveyors

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ListSurveyors lists the agrimensores. Only callers with the administrador
// role may do this, since the ABM they feed is administrador only.
type ListSurveyors interface {
	Execute(ctx context.Context, actorRoles []string, includeInactive bool) ([]domain.Usuario, error)
}

type listSurveyorsUseCase struct {
	repository gateway.UserRepository
}

func NewListSurveyors(repository gateway.UserRepository) ListSurveyors {
	return &listSurveyorsUseCase{repository: repository}
}

func (useCase *listSurveyorsUseCase) Execute(
	ctx context.Context,
	actorRoles []string,
	includeInactive bool,
) ([]domain.Usuario, error) {
	if !domain.HasRole(actorRoles, domain.RolAdministrador) {
		return nil, domain.ErrNoAutorizado
	}

	agrimensores, err := useCase.repository.ListByRol(ctx, domain.RolAgrimensor, includeInactive)
	if err != nil {
		return nil, fromRepository(err)
	}

	return agrimensores, nil
}
