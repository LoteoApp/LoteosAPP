package agencies

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type ListAgenciesInput struct {
	ActorRoles []string
	Search     string
}

// ListAgencies searches active inmobiliarias by razón social or CUIT. The
// catálogo is what an administrativo picks from when operating a loteo, so
// listing is open to administrador and administrativo; only administrador
// may change it.
type ListAgencies interface {
	Execute(ctx context.Context, input ListAgenciesInput) ([]domain.Inmobiliaria, error)
}

type listAgenciesUseCase struct {
	repository gateway.InmobiliariaRepository
}

func NewListAgencies(repository gateway.InmobiliariaRepository) ListAgencies {
	return &listAgenciesUseCase{repository: repository}
}

func (useCase *listAgenciesUseCase) Execute(ctx context.Context, input ListAgenciesInput) ([]domain.Inmobiliaria, error) {
	if !hasRole(input.ActorRoles, domain.RolAdministrador, domain.RolAdministrativo) {
		return nil, domain.ErrNoAutorizado
	}

	// A CUIT is stored without separators, so a search typed as
	// "30-71234567-8" has to be normalized before it reaches the LIKE.
	search := strings.TrimSpace(input.Search)
	if normalized := domain.NormalizarCUIT(search); domain.CUITValido(normalized) {
		search = normalized
	}

	inmobiliarias, err := useCase.repository.List(ctx, search)
	if err != nil {
		return nil, fromRepository(err)
	}

	return inmobiliarias, nil
}
