package surveyors

import (
	"context"
	"errors"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestDeactivateSurveyorRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: activeSurveyor()}
	deactivateSurveyor := NewDeactivateSurveyor(repository)

	err := deactivateSurveyor.Execute(context.Background(), []string{domain.RolAdministrativo}, "admin-sub", "agri-1")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not give anyone de baja when actor is not administrador")
	}
}

func TestDeactivateSurveyorHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:             activeSurveyor(),
		FoundByAuthProviderID: domain.Usuario{ID: "admin-1", Rol: domain.RolAdministrador},
	}
	deactivateSurveyor := NewDeactivateSurveyor(repository)

	if err := deactivateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "agri-1"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.SoftDeletedID != "agri-1" {
		t.Errorf("Execute() gave de baja %q, want %q", repository.SoftDeletedID, "agri-1")
	}
	if repository.SoftDeletedActor != "admin-1" {
		t.Errorf("Execute() usuario_modificacion = %q, want %q", repository.SoftDeletedActor, "admin-1")
	}
}

func TestDeactivateSurveyorRejectsUnknownID(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByIDErr: domain.ErrUsuarioNoEncontrado}
	deactivateSurveyor := NewDeactivateSurveyor(repository)

	err := deactivateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "agri-1")

	if !errors.Is(err, domain.ErrAgrimensorNoEncontrado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgrimensorNoEncontrado)
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not give de baja a user it could not find")
	}
}

func TestDeactivateSurveyorRejectsUserOfAnotherRol(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID: domain.Usuario{ID: "admin-2", Rol: domain.RolAdministrador},
	}
	deactivateSurveyor := NewDeactivateSurveyor(repository)

	err := deactivateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "admin-2")

	if !errors.Is(err, domain.ErrAgrimensorNoEncontrado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgrimensorNoEncontrado)
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not give de baja a user that is not an agrimensor")
	}
}

func TestDeactivateSurveyorRejectsAlreadyInactive(t *testing.T) {
	t.Parallel()

	baja := time.Now()
	inactive := activeSurveyor()
	inactive.FechaBaja = &baja
	repository := &gatewayfake.UserRepository{FoundByID: inactive}
	deactivateSurveyor := NewDeactivateSurveyor(repository)

	err := deactivateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "agri-1")

	if !errors.Is(err, domain.ErrAgrimensorDadoDeBaja) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgrimensorDadoDeBaja)
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not repeat the baja of an inactive agrimensor")
	}
}

func TestDeactivateSurveyorWrapsActorLookupFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:               activeSurveyor(),
		FindByAuthProviderIDErr: errors.New("connection refused"),
	}
	deactivateSurveyor := NewDeactivateSurveyor(repository)

	err := deactivateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "agri-1")

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not give de baja when the actor cannot be resolved")
	}
}

func TestDeactivateSurveyorWrapsSoftDeleteFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:     activeSurveyor(),
		SoftDeleteErr: errors.New("connection refused"),
	}
	deactivateSurveyor := NewDeactivateSurveyor(repository)

	err := deactivateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "agri-1")

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
}

func TestDeactivateSurveyorWrapsUnexpectedLookupFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByIDErr: errors.New("connection refused")}
	deactivateSurveyor := NewDeactivateSurveyor(repository)

	err := deactivateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "agri-1")

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
}
