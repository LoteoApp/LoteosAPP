package agencies

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func ptr(s string) *string {
	return &s
}

func TestCreateAgencyRejectsUnauthorizedRole(t *testing.T) {
	t.Parallel()

	roles := []string{domain.RolAdministrativo, domain.RolAgrimensor, domain.RolEscribano, domain.RolInmobiliaria}

	for _, rol := range roles {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.InmobiliariaRepository{}
			users := &gatewayfake.UserRepository{}
			createAgency := NewCreateAgency(repository, users)

			_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
				ActorRoles: []string{rol}, Subject: "sb-1", RazonSocial: "Lotes del Sur",
			})

			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if repository.CreateCalls != 0 {
				t.Error("Execute() should not call repository when actor is not authorized")
			}
		})
	}
}

func TestCreateAgencyRejectsEmptyRazonSocial(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", RazonSocial: "   ",
	})

	if !errors.Is(err, domain.ErrInmobiliariaInvalida) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInmobiliariaInvalida)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not call repository with invalid input")
	}
}

func TestCreateAgencyRejectsInvalidCUIT(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cuit string
	}{
		{name: "too short", cuit: "3071234567"},
		{name: "letters", cuit: "30-7123456x-8"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.InmobiliariaRepository{}
			users := &gatewayfake.UserRepository{}
			createAgency := NewCreateAgency(repository, users)

			_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
				ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
				RazonSocial: "Lotes del Sur", CUIT: ptr(test.cuit),
			})

			if !errors.Is(err, domain.ErrCUITInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrCUITInvalido)
			}
			if repository.CreateCalls != 0 {
				t.Error("Execute() should not call repository with an invalid cuit")
			}
		})
	}
}

func TestCreateAgencyRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		RazonSocial: "Lotes del Sur", Email: ptr("contacto.example.com"),
	})

	if !errors.Is(err, domain.ErrEmailInvalido) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailInvalido)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not call repository with an invalid email")
	}
}

func TestCreateAgencyHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createAgency := NewCreateAgency(repository, users)

	inmobiliaria, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		RazonSocial: "  Lotes del Sur  ", CUIT: ptr(" 30-71234567-8 "),
		Telefono: ptr(" 3415551234 "), Email: ptr("  contacto@lotesdelsur.com  "),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if inmobiliaria.RazonSocial != "Lotes del Sur" {
		t.Errorf("Execute() razon social = %q, want it trimmed", inmobiliaria.RazonSocial)
	}
	if inmobiliaria.CUIT == nil || *inmobiliaria.CUIT != "30712345678" {
		t.Errorf("Execute() cuit = %v, want it normalized to digits", inmobiliaria.CUIT)
	}
	if inmobiliaria.Telefono == nil || *inmobiliaria.Telefono != "3415551234" {
		t.Errorf("Execute() telefono = %v, want it trimmed", inmobiliaria.Telefono)
	}
	if inmobiliaria.Email == nil || *inmobiliaria.Email != "contacto@lotesdelsur.com" {
		t.Errorf("Execute() email = %v, want it trimmed", inmobiliaria.Email)
	}
	if inmobiliaria.UsuarioModificacion != "user-1" {
		t.Errorf("Execute() usuario modificacion = %q, want %q", inmobiliaria.UsuarioModificacion, "user-1")
	}
	if repository.CreateCalls != 1 {
		t.Errorf("Execute() repository.Create calls = %d, want 1", repository.CreateCalls)
	}
	if users.FindByAuthProviderIDSubject != "sb-1" {
		t.Errorf("Execute() resolved subject = %q, want %q", users.FindByAuthProviderIDSubject, "sb-1")
	}
}

func TestCreateAgencyDropsBlankOptionalFields(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createAgency := NewCreateAgency(repository, users)

	inmobiliaria, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		RazonSocial: "Lotes del Sur", CUIT: ptr("  "), Telefono: ptr(""), Email: ptr("   "),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if inmobiliaria.CUIT != nil || inmobiliaria.Telefono != nil || inmobiliaria.Email != nil {
		t.Errorf("Execute() = %#v, want every blank optional field absent", inmobiliaria)
	}
}

func TestCreateAgencyPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrCUITEnUso
	repository := &gatewayfake.InmobiliariaRepository{CreateErr: wantErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", RazonSocial: "Lotes del Sur",
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestCreateAgencyClassifiesUnexpectedRepositoryError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("syntax error at end of input")
	repository := &gatewayfake.InmobiliariaRepository{CreateErr: rawErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", RazonSocial: "Lotes del Sur",
	})

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
	if !errors.Is(err, rawErr) {
		t.Errorf("Execute() error = %v, want it to carry %v as Cause for the log", err, rawErr)
	}
}

func TestCreateAgencyRejectsActorWithoutUsuario(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", RazonSocial: "Lotes del Sur",
	})

	if !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}

func TestCreateAgencyClassifiesActorResolutionError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("connection refused")
	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: rawErr}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", RazonSocial: "Lotes del Sur",
	})

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
	if !errors.Is(err, rawErr) {
		t.Errorf("Execute() error = %v, want it to carry %v as Cause for the log", err, rawErr)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}
