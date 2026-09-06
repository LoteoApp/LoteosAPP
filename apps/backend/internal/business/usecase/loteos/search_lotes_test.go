package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func TestSearchLotesLetsUnrestrictedRolesReachEveryLote(t *testing.T) {
	t.Parallel()

	for _, role := range []string{domain.RolAdministrador, domain.RolAdministrativo} {
		t.Run(role, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.LoteoRepository{
				SearchLotesResult: []domain.LoteSummary{{ID: "lote-1", Number: "7", LoteoName: "Norte"}},
			}
			useCase := loteos.NewSearchLotes(repository)

			got, err := useCase.Execute(context.Background(), loteos.SearchLotesInput{
				Actor: actorWith(role), Search: "  7  ",
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(got) != 1 || got[0].ID != "lote-1" {
				t.Errorf("Execute() = %#v", got)
			}
			if repository.SearchLotesScope.AssigneeAuthProviderID != nil {
				t.Errorf(
					"assignee = %v, want nil so every lote is reachable",
					*repository.SearchLotesScope.AssigneeAuthProviderID,
				)
			}
			if repository.SearchLotesFilter.Search != "7" {
				t.Errorf("search = %q, want it trimmed", repository.SearchLotesFilter.Search)
			}
		})
	}
}

func TestSearchLotesScopesInmobiliariaToItsAgencyLoteos(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewSearchLotes(repository)

	if _, err := useCase.Execute(context.Background(), loteos.SearchLotesInput{
		Actor: actorWith(domain.RolInmobiliaria),
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	scope := repository.SearchLotesScope
	if scope.AssigneeAuthProviderID == nil || *scope.AssigneeAuthProviderID != "actor-1" {
		t.Errorf("assignee = %v, want the caller", scope.AssigneeAuthProviderID)
	}
	if !scope.ByAgencyAssignment {
		t.Error("ByAgencyAssignment = false, want the agency path enabled")
	}
	if scope.ByUserAssignment {
		t.Error("ByUserAssignment = true, want an inmobiliaria limited to its agency loteos")
	}
}

func TestSearchLotesRejectsRolesThatDoNotCarryASale(t *testing.T) {
	t.Parallel()

	for _, role := range []string{domain.RolAgrimensor, domain.RolEscribano} {
		t.Run(role, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.LoteoRepository{}
			useCase := loteos.NewSearchLotes(repository)

			_, err := useCase.Execute(context.Background(), loteos.SearchLotesInput{
				Actor: actorWith(role),
			})
			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if repository.SearchLotesCalls != 0 {
				t.Errorf("SearchLotesCalls = %d, want the repository untouched", repository.SearchLotesCalls)
			}
		})
	}
}

func TestSearchLotesReportsARepositoryFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{SearchLotesErr: errors.New("boom")}
	useCase := loteos.NewSearchLotes(repository)

	_, err := useCase.Execute(context.Background(), loteos.SearchLotesInput{
		Actor: actorWith(domain.RolAdministrador),
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want the repository failure reported")
	}
}

func TestSearchLotesPassesTheRequestedStateToTheRepository(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewSearchLotes(repository)

	if _, err := useCase.Execute(context.Background(), loteos.SearchLotesInput{
		Actor: actorWith(domain.RolAdministrador), State: domain.LotStateAvailable,
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if repository.SearchLotesFilter.State != domain.LotStateAvailable {
		t.Errorf("state = %q, want %q", repository.SearchLotesFilter.State, domain.LotStateAvailable)
	}
}

func TestSearchLotesWithoutAStateLeavesEveryStateReachable(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewSearchLotes(repository)

	if _, err := useCase.Execute(context.Background(), loteos.SearchLotesInput{
		Actor: actorWith(domain.RolAdministrador),
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if repository.SearchLotesFilter.State != "" {
		t.Errorf("state = %q, want it empty so every state is listed", repository.SearchLotesFilter.State)
	}
}

func TestSearchLotesRejectsAnUnknownState(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewSearchLotes(repository)

	_, err := useCase.Execute(context.Background(), loteos.SearchLotesInput{
		Actor: actorWith(domain.RolAdministrador), State: domain.LotState("rifado"),
	})
	if !errors.Is(err, domain.ErrInvalidLotState) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidLotState)
	}
	if repository.SearchLotesCalls != 0 {
		t.Errorf("SearchLotesCalls = %d, want the repository untouched", repository.SearchLotesCalls)
	}
}
