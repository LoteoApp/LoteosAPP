package loteos_test

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/loteos"
)

func TestGetLoteoLetsUnrestrictedRolesReadAnyLoteo(t *testing.T) {
	t.Parallel()

	for _, role := range []string{domain.RolAdministrador, domain.RolAdministrativo} {
		t.Run(role, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.LoteoRepository{
				GetResult: domain.Loteo{ID: "loteo-1", Name: "Norte"},
			}
			useCase := loteos.NewGetLoteo(repository)

			got, err := useCase.Execute(context.Background(), actorWith(role), "  loteo-1  ")
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got.ID != "loteo-1" {
				t.Errorf("Execute() = %#v", got)
			}
			if repository.GetLoteoID != "loteo-1" {
				t.Errorf("loteo id = %q, want it trimmed", repository.GetLoteoID)
			}
			if repository.GetScope.AssigneeAuthProviderID != nil {
				t.Errorf("assignee = %v, want nil", *repository.GetScope.AssigneeAuthProviderID)
			}
		})
	}
}

func TestGetLoteoScopesAssignedRolesToTheirOwnAssignmentPath(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		role         string
		wantByUser   bool
		wantByAgency bool
	}{
		"agrimensor":   {role: domain.RolAgrimensor, wantByUser: true, wantByAgency: false},
		"escribano":    {role: domain.RolEscribano, wantByUser: true, wantByAgency: false},
		"inmobiliaria": {role: domain.RolInmobiliaria, wantByUser: false, wantByAgency: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.LoteoRepository{}
			useCase := loteos.NewGetLoteo(repository)

			if _, err := useCase.Execute(context.Background(), actorWith(test.role), "loteo-1"); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			scope := repository.GetScope
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

func TestGetLoteoDeniesAnyOtherActor(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewGetLoteo(repository)

	_, err := useCase.Execute(context.Background(), loteos.Actor{AuthProviderID: "actor-1"}, "loteo-1")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.GetCalls != 0 {
		t.Error("Execute() should not reach the repository for an unauthorized actor")
	}
}

func TestGetLoteoTreatsAnEmptyIDAsNotFound(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{}
	useCase := loteos.NewGetLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "   ")

	if !errors.Is(err, domain.ErrLoteoNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteoNotFound)
	}
	if repository.GetCalls != 0 {
		t.Error("Execute() should not reach the repository for an empty id")
	}
}

func TestGetLoteoPropagatesNotFound(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.LoteoRepository{GetErr: domain.ErrLoteoNotFound}
	useCase := loteos.NewGetLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1")

	if !errors.Is(err, domain.ErrLoteoNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrLoteoNotFound)
	}
}

func TestGetLoteoHidesAnUnexpectedRepositoryError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("i/o timeout")
	repository := &gatewayfake.LoteoRepository{GetErr: rawErr}
	useCase := loteos.NewGetLoteo(repository)

	_, err := useCase.Execute(context.Background(), administrador(), "loteo-1")

	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != domain.KindUnavailable {
		t.Fatalf("Execute() error = %v, want an unavailable *domain.Error", err)
	}
	if !errors.Is(err, rawErr) {
		t.Error("Execute() should keep the underlying error as the cause")
	}
}
