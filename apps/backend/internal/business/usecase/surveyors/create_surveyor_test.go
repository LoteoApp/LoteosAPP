package surveyors

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestCreateSurveyorRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{}
	createSurveyor := NewCreateSurveyor(repository, identity)

	_, _, err := createSurveyor.Execute(context.Background(), []string{domain.RolAdministrativo}, "Ana", "Gómez", "ana@example.com")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if identity.CreateCalls != 0 || repository.CreateCalls != 0 {
		t.Error("Execute() should not touch any gateway when actor is not administrador")
	}
}

func TestCreateSurveyorRejectsIncompleteProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nombre   string
		apellido string
	}{
		{name: "sin nombre", apellido: "Gómez"},
		{name: "sin apellido", nombre: "Ana"},
		{name: "solo espacios", nombre: "   ", apellido: "   "},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.UserRepository{}
			identity := &gatewayfake.IdentityProvider{}
			createSurveyor := NewCreateSurveyor(repository, identity)

			_, _, err := createSurveyor.Execute(context.Background(),
				[]string{domain.RolAdministrador}, test.nombre, test.apellido, "ana@example.com")

			if !errors.Is(err, domain.ErrPerfilInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrPerfilInvalido)
			}
			if identity.CreateCalls != 0 {
				t.Error("Execute() should not call the identity provider with an incomplete profile")
			}
		})
	}
}

func TestCreateSurveyorRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	for _, email := range []string{"", "   ", "not-an-email"} {
		t.Run(email, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.UserRepository{}
			identity := &gatewayfake.IdentityProvider{}
			createSurveyor := NewCreateSurveyor(repository, identity)

			_, _, err := createSurveyor.Execute(context.Background(),
				[]string{domain.RolAdministrador}, "Ana", "Gómez", email)

			if !errors.Is(err, domain.ErrEmailInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailInvalido)
			}
			if identity.CreateCalls != 0 {
				t.Error("Execute() should not call the identity provider with an invalid email")
			}
		})
	}
}

func TestCreateSurveyorHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", TempPassword: "temp-pass-123"}
	createSurveyor := NewCreateSurveyor(repository, identity)

	agrimensor, tempPassword, err := createSurveyor.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "  Ana  ", "  Gómez  ", "  ana@example.com  ")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if agrimensor.Rol != domain.RolAgrimensor {
		t.Errorf("Execute() rol = %q, want %q", agrimensor.Rol, domain.RolAgrimensor)
	}
	if agrimensor.Nombre != "Ana" || agrimensor.Apellido != "Gómez" {
		t.Errorf("Execute() should trim the profile, got %q %q", agrimensor.Nombre, agrimensor.Apellido)
	}
	if agrimensor.Email != "ana@example.com" {
		t.Errorf("Execute() email = %q, want %q", agrimensor.Email, "ana@example.com")
	}
	if !agrimensor.PerfilCompleto {
		t.Error("Execute() should mark the profile as complete when nombre and apellido are given")
	}
	if !agrimensor.Activo() {
		t.Error("Execute() should create an active agrimensor")
	}
	if tempPassword != "temp-pass-123" {
		t.Errorf("Execute() temporary password = %q, want %q", tempPassword, "temp-pass-123")
	}
	if identity.DeleteCalls != 0 {
		t.Error("Execute() should not compensate when persistence succeeds")
	}
}

func TestCreateSurveyorPropagatesIdentityProviderConflict(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{CreateErr: domain.ErrEmailEnUso}
	createSurveyor := NewCreateSurveyor(repository, identity)

	_, _, err := createSurveyor.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com")

	if !errors.Is(err, domain.ErrEmailEnUso) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailEnUso)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not persist when the identity provider fails")
	}
}

func TestCreateSurveyorCompensatesWhenPersistenceFails(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{CreateErr: errors.New("insert failed")}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123"}
	createSurveyor := NewCreateSurveyor(repository, identity)

	_, _, err := createSurveyor.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com")

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
	if identity.DeleteCalls != 1 || identity.DeletedUserID != "sb-123" {
		t.Errorf("Execute() should delete the orphaned account, calls = %d id = %q",
			identity.DeleteCalls, identity.DeletedUserID)
	}
}

func TestCreateSurveyorReturnsConflictWhenEmailAlreadyPersisted(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{CreateErr: domain.ErrEmailEnUso}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123"}
	createSurveyor := NewCreateSurveyor(repository, identity)

	_, _, err := createSurveyor.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com")

	if !errors.Is(err, domain.ErrEmailEnUso) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailEnUso)
	}
	if identity.DeleteCalls != 1 {
		t.Errorf("Execute() identity.DeleteUser calls = %d, want 1", identity.DeleteCalls)
	}
}

func TestCreateSurveyorReturnsOriginalErrorWhenCompensationAlsoFails(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{CreateErr: domain.ErrEmailEnUso}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", DeleteErr: errors.New("delete failed")}
	createSurveyor := NewCreateSurveyor(repository, identity)

	_, _, err := createSurveyor.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com")

	if !errors.Is(err, domain.ErrEmailEnUso) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailEnUso)
	}
}
