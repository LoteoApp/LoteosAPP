package agencies

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestListAgenciesRejectsUnauthorizedRole(t *testing.T) {
	t.Parallel()

	roles := []string{domain.RolAgrimensor, domain.RolEscribano, domain.RolInmobiliaria}

	for _, rol := range roles {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.AgencyRepository{}
			listAgencies := NewListAgencies(repository)

			_, err := listAgencies.Execute(context.Background(), ListAgenciesInput{ActorRoles: []string{rol}})

			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if repository.ListCalls != 0 {
				t.Error("Execute() should not call repository when actor is not authorized")
			}
		})
	}
}

func TestListAgenciesAllowsAdministradorAndAdministrativo(t *testing.T) {
	t.Parallel()

	roles := []string{domain.RolAdministrador, domain.RolAdministrativo}

	for _, rol := range roles {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.AgencyRepository{
				ListResult: []domain.Agency{{ID: "agency-1", BusinessName: "Lotes del Sur"}},
			}
			listAgencies := NewListAgencies(repository)

			found, err := listAgencies.Execute(context.Background(), ListAgenciesInput{ActorRoles: []string{rol}})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(found) != 1 || found[0].ID != "agency-1" {
				t.Errorf("Execute() = %#v", found)
			}
			if repository.ListCalls != 1 {
				t.Errorf("Execute() repository.List calls = %d, want 1", repository.ListCalls)
			}
		})
	}
}

func TestListAgenciesTrimsTheSearch(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.AgencyRepository{}
	listAgencies := NewListAgencies(repository)

	if _, err := listAgencies.Execute(context.Background(), ListAgenciesInput{
		ActorRoles: []string{domain.RolAdministrador}, Search: "  sur  ",
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.ListSearch != "sur" {
		t.Errorf("Execute() search = %q, want %q", repository.ListSearch, "sur")
	}
}

// A CUIT is stored without separators, so searching it the way it is printed
// has to find the inmobiliaria anyway.
func TestListAgenciesNormalizesASearchedCUIT(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.AgencyRepository{}
	listAgencies := NewListAgencies(repository)

	if _, err := listAgencies.Execute(context.Background(), ListAgenciesInput{
		ActorRoles: []string{domain.RolAdministrador}, Search: " 30-71234567-8 ",
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.ListSearch != "30712345678" {
		t.Errorf("Execute() search = %q, want %q", repository.ListSearch, "30712345678")
	}
}

func TestListAgenciesLeavesANameSearchAlone(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.AgencyRepository{}
	listAgencies := NewListAgencies(repository)

	if _, err := listAgencies.Execute(context.Background(), ListAgenciesInput{
		ActorRoles: []string{domain.RolAdministrador}, Search: "Lotes del Sur",
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.ListSearch != "Lotes del Sur" {
		t.Errorf("Execute() search = %q, want it untouched", repository.ListSearch)
	}
}

func TestListAgenciesClassifiesUnexpectedRepositoryError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("connection refused")
	repository := &gatewayfake.AgencyRepository{ListErr: rawErr}
	listAgencies := NewListAgencies(repository)

	_, err := listAgencies.Execute(context.Background(), ListAgenciesInput{ActorRoles: []string{domain.RolAdministrador}})

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
	if !errors.Is(err, rawErr) {
		t.Errorf("Execute() error = %v, want it to carry %v as Cause for the log", err, rawErr)
	}
}

// A *domain.Error the repository already classified travels unchanged: the
// caller still gets the specific conflict, not a generic unavailable.
func TestListAgenciesKeepsDomainErrors(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.AgencyRepository{ListErr: domain.ErrNoAutorizado}
	listAgencies := NewListAgencies(repository)

	_, err := listAgencies.Execute(context.Background(), ListAgenciesInput{ActorRoles: []string{domain.RolAdministrador}})

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Error("Execute() should not reclassify an error the gateway already classified")
	}
}
