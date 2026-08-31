package surveyors

import (
	"context"
	"errors"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestListSurveyorsRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	listSurveyors := NewListSurveyors(repository)

	_, err := listSurveyors.Execute(context.Background(), []string{domain.RolAgrimensor}, false)

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.ListByRolCalls != 0 {
		t.Error("Execute() should not query the repository when actor is not administrador")
	}
}

func TestListSurveyorsReturnsOnlyActiveByDefault(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		ListByRolResult: []domain.Usuario{{ID: "u-1", Rol: domain.RolAgrimensor}},
	}
	listSurveyors := NewListSurveyors(repository)

	agrimensores, err := listSurveyors.Execute(context.Background(), []string{domain.RolAdministrador}, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(agrimensores) != 1 || agrimensores[0].ID != "u-1" {
		t.Errorf("Execute() = %#v", agrimensores)
	}
	if repository.ListByRolInput != domain.RolAgrimensor {
		t.Errorf("Execute() queried rol %q, want %q", repository.ListByRolInput, domain.RolAgrimensor)
	}
	if repository.ListByRolInactive {
		t.Error("Execute() should not include inactive agrimensores by default")
	}
}

func TestListSurveyorsIncludesInactiveWhenAsked(t *testing.T) {
	t.Parallel()

	baja := time.Now()
	repository := &gatewayfake.UserRepository{
		ListByRolResult: []domain.Usuario{
			{ID: "u-1", Rol: domain.RolAgrimensor},
			{ID: "u-2", Rol: domain.RolAgrimensor, FechaBaja: &baja},
		},
	}
	listSurveyors := NewListSurveyors(repository)

	agrimensores, err := listSurveyors.Execute(context.Background(), []string{domain.RolAdministrador}, true)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(agrimensores) != 2 {
		t.Fatalf("Execute() returned %d agrimensores, want 2", len(agrimensores))
	}
	if agrimensores[1].Activo() {
		t.Error("Execute() should keep the fecha de baja of an inactive agrimensor")
	}
	if !repository.ListByRolInactive {
		t.Error("Execute() should ask the repository for the inactive agrimensores too")
	}
}

func TestListSurveyorsWrapsRepositoryFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{ListByRolErr: errors.New("connection refused")}
	listSurveyors := NewListSurveyors(repository)

	_, err := listSurveyors.Execute(context.Background(), []string{domain.RolAdministrador}, false)

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
}
