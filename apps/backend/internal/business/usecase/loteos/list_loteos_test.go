package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func actorWith(role string) loteos.Actor {
	return loteos.Actor{AuthProviderID: "actor-1", Roles: []string{role}}
}

func TestListLoteosLetsUnrestrictedRolesSeeEveryLoteo(t *testing.T) {
	t.Parallel()

	for _, role := range []string{domain.RolAdministrador, domain.RolAdministrativo} {
		t.Run(role, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.LoteoRepository{
				ListResult: []domain.LoteoSummary{{ID: "loteo-1", Name: "Norte"}},
			}
			useCase := loteos.NewListLoteos(repository)

			got, err := useCase.Execute(context.Background(), loteos.ListLoteosInput{
				Actor: actorWith(role), Search: "  norte  ",
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(got) != 1 || got[0].ID != "loteo-1" {
				t.Errorf("Execute() = %#v", got)
			}
			if repository.ListScope.AssigneeAuthProviderID != nil {
				t.Errorf("assignee = %v, want nil so every loteo is listed", *repository.ListScope.AssigneeAuthProviderID)
			}
			if repository.ListSearch != "norte" {
				t.Errorf("search = %q, want it trimmed", repository.ListSearch)
			}
		})
	}
}

func TestListLoteosScopesAssignedRolesToTheirOwnAssignmentPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		role         string
		wantByUser   bool
		wantByAgency bool
	}{
		"agrimensor reaches loteos through usuario_loteos only": {
			role: domain.RolAgrimensor, wantByUser: true, wantByAgency: false,
		},
		"escribano reaches loteos through usuario_loteos only": {
			role: domain.RolEscribano, wantByUser: true, wantByAgency: false,
		},
		"inmobiliaria reaches loteos through its agency only": {
			role: domain.RolInmobiliaria, wantByUser: false, wantByAgency: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.LoteoRepository{}
			useCase := loteos.NewListLoteos(repository)

			if _, err := useCase.Execute(context.Background(), loteos.ListLoteosInput{
				Actor: actorWith(test.role),
			}); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			scope := repository.ListScope
			if scope.AssigneeAuthProviderID == nil || *scope.AssigneeAuthProviderID != "actor-1" {
				t.Errorf("assignee = %v, want the actor's id", scope.AssigneeAuthProviderID)
			}
			if scope.ByUserAssignment != test.wantByUser || scope.ByAgencyAssignment != test.wantByAgency {
				t.Errorf("scope = {byUser: %v, byAgency: %v}, want {byUser: %v, byAgency: %v}",
					scope.ByUserAssignment, scope.ByAgencyAssignment, test.wantByUser, test.wantByAgency)
			}
		})
	}
}

func TestListLoteosGivesAMultiRoleActorTheUnionOfItsPaths(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewListLoteos(repository)

	if _, err := useCase.Execute(context.Background(), loteos.ListLoteosInput{
		Actor: loteos.Actor{AuthProviderID: "actor-1", Roles: []string{domain.RolAgrimensor, domain.RolInmobiliaria}},
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	scope := repository.ListScope
	if !scope.ByUserAssignment || !scope.ByAgencyAssignment {
		t.Errorf("scope = %#v, want both assignment paths enabled", scope)
	}
}

func TestListLoteosDeniesAnyOtherActor(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewListLoteos(repository)

	_, err := useCase.Execute(context.Background(), loteos.ListLoteosInput{
		Actor: loteos.Actor{AuthProviderID: "actor-1", Roles: nil},
	})

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.ListCalls != 0 {
		t.Error("Execute() should not reach the repository for an unauthorized actor")
	}
}

func TestListLoteosPropagatesABusinessError(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{ListErr: domain.ErrLoteoNotFound}
	useCase := loteos.NewListLoteos(repository)

	_, err := useCase.Execute(context.Background(), loteos.ListLoteosInput{Actor: administrador()})

	if !errors.Is(err, domain.ErrLoteoNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteoNotFound)
	}
}

func TestListLoteosHidesAnUnexpectedRepositoryError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("connection reset by peer")
	repository := &gatewayfake.LoteoRepository{ListErr: rawErr}
	useCase := loteos.NewListLoteos(repository)

	_, err := useCase.Execute(context.Background(), loteos.ListLoteosInput{Actor: administrador()})

	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != domain.KindUnavailable {
		t.Fatalf("Execute() error = %v, want an unavailable *domain.Error", err)
	}
	if !errors.Is(err, rawErr) {
		t.Error("Execute() should keep the underlying error as the cause")
	}
}
