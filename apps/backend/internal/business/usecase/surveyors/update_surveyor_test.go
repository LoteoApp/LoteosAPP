package surveyors

import (
	"context"
	"errors"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func stringPtr(value string) *string {
	return &value
}

func activeSurveyor() domain.Usuario {
	return domain.Usuario{ID: "agri-1", Rol: domain.RolAgrimensor, Nombre: "Ana", Apellido: "Gómez"}
}

func TestUpdateSurveyorRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: activeSurveyor()}
	updateSurveyor := NewUpdateSurveyor(repository)

	_, err := updateSurveyor.Execute(context.Background(), []string{domain.RolAgrimensor},
		"admin-sub", "agri-1", stringPtr("Ana María"), nil)

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not update when actor is not administrador")
	}
}

func TestUpdateSurveyorRejectsBlankProfileFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nombre   *string
		apellido *string
	}{
		{name: "nombre en blanco", nombre: stringPtr("   ")},
		{name: "apellido en blanco", apellido: stringPtr("")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.UserRepository{FoundByID: activeSurveyor()}
			updateSurveyor := NewUpdateSurveyor(repository)

			_, err := updateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador},
				"admin-sub", "agri-1", test.nombre, test.apellido)

			if !errors.Is(err, domain.ErrPerfilInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrPerfilInvalido)
			}
			if repository.UpdateCalls != 0 {
				t.Error("Execute() should not update with a blank profile field")
			}
		})
	}
}

func TestUpdateSurveyorTrimsAndPersistsOnlyTheFieldsSent(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:                  activeSurveyor(),
		FindByAuthProviderIDResult: domain.Usuario{ID: "admin-1", Rol: domain.RolAdministrador},
	}
	updateSurveyor := NewUpdateSurveyor(repository)

	updated, err := updateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador},
		"admin-sub", "agri-1", stringPtr("  Ana María  "), nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if updated.Nombre != "Ana María" {
		t.Errorf("Execute() nombre = %q, want %q", updated.Nombre, "Ana María")
	}
	if repository.UpdateInput.Apellido != nil {
		t.Error("Execute() should leave an omitted field as unchanged")
	}
	if repository.UpdateInput.UsuarioModificacion != "admin-1" {
		t.Errorf("Execute() usuario_modificacion = %q, want %q", repository.UpdateInput.UsuarioModificacion, "admin-1")
	}
}

func TestUpdateSurveyorRejectsUnknownID(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByIDErr: domain.ErrUsuarioNoEncontrado}
	updateSurveyor := NewUpdateSurveyor(repository)

	_, err := updateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador},
		"admin-sub", "agri-1", stringPtr("Ana"), nil)

	if !errors.Is(err, domain.ErrAgrimensorNoEncontrado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgrimensorNoEncontrado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not update a user it could not find")
	}
}

func TestUpdateSurveyorRejectsUserOfAnotherRol(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID: domain.Usuario{ID: "admin-2", Rol: domain.RolAdministrador},
	}
	updateSurveyor := NewUpdateSurveyor(repository)

	_, err := updateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador},
		"admin-sub", "admin-2", stringPtr("Ana"), nil)

	if !errors.Is(err, domain.ErrAgrimensorNoEncontrado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgrimensorNoEncontrado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not update a user that is not an agrimensor")
	}
}

func TestUpdateSurveyorRejectsInactiveSurveyor(t *testing.T) {
	t.Parallel()

	baja := time.Now()
	inactive := activeSurveyor()
	inactive.FechaBaja = &baja
	repository := &gatewayfake.UserRepository{FoundByID: inactive}
	updateSurveyor := NewUpdateSurveyor(repository)

	_, err := updateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador},
		"admin-sub", "agri-1", stringPtr("Ana"), nil)

	if !errors.Is(err, domain.ErrAgrimensorNoEncontrado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgrimensorNoEncontrado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not update an agrimensor given de baja")
	}
}

func TestUpdateSurveyorWrapsActorLookupFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:               activeSurveyor(),
		FindByAuthProviderIDErr: errors.New("connection refused"),
	}
	updateSurveyor := NewUpdateSurveyor(repository)

	_, err := updateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador},
		"admin-sub", "agri-1", stringPtr("Ana"), nil)

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
}

func TestUpdateSurveyorWrapsUpdateFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID: activeSurveyor(),
		UpdateErr: errors.New("connection refused"),
	}
	updateSurveyor := NewUpdateSurveyor(repository)

	_, err := updateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador},
		"admin-sub", "agri-1", stringPtr("Ana"), nil)

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
}

func TestUpdateSurveyorWrapsUnexpectedLookupFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByIDErr: errors.New("connection refused")}
	updateSurveyor := NewUpdateSurveyor(repository)

	_, err := updateSurveyor.Execute(context.Background(), []string{domain.RolAdministrador},
		"admin-sub", "agri-1", stringPtr("Ana"), nil)

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
}
