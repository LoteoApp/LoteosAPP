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

			repository := &gatewayfake.AgencyRepository{}
			users := &gatewayfake.UserRepository{}
			createAgency := NewCreateAgency(repository, users)

			_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
				ActorRoles: []string{rol}, Subject: "sb-1", BusinessName: "Lotes del Sur",
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

	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", BusinessName: "   ",
	})

	if !errors.Is(err, domain.ErrInvalidAgency) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidAgency)
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

			repository := &gatewayfake.AgencyRepository{}
			users := &gatewayfake.UserRepository{}
			createAgency := NewCreateAgency(repository, users)

			_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
				ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
				BusinessName: "Lotes del Sur", CUIT: ptr(test.cuit),
			})

			if !errors.Is(err, domain.ErrInvalidCUIT) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidCUIT)
			}
			if repository.CreateCalls != 0 {
				t.Error("Execute() should not call repository with an invalid cuit")
			}
		})
	}
}

func TestCreateAgencyRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		BusinessName: "Lotes del Sur", Email: ptr("contacto.example.com"),
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

	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createAgency := NewCreateAgency(repository, users)

	agency, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		BusinessName: "  Lotes del Sur  ", CUIT: ptr(" 30-71234567-8 "),
		Phone: ptr(" 3415551234 "), Email: ptr("  contacto@lotesdelsur.com  "),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if agency.BusinessName != "Lotes del Sur" {
		t.Errorf("Execute() razon social = %q, want it trimmed", agency.BusinessName)
	}
	if agency.CUIT == nil || *agency.CUIT != "30712345678" {
		t.Errorf("Execute() cuit = %v, want it normalized to digits", agency.CUIT)
	}
	if agency.Phone == nil || *agency.Phone != "3415551234" {
		t.Errorf("Execute() telefono = %v, want it trimmed", agency.Phone)
	}
	if agency.Email == nil || *agency.Email != "contacto@lotesdelsur.com" {
		t.Errorf("Execute() email = %v, want it trimmed", agency.Email)
	}
	if agency.ModifiedBy != "user-1" {
		t.Errorf("Execute() usuario modificacion = %q, want %q", agency.ModifiedBy, "user-1")
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

	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createAgency := NewCreateAgency(repository, users)

	agency, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		BusinessName: "Lotes del Sur", CUIT: ptr("  "), Phone: ptr(""), Email: ptr("   "),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if agency.CUIT != nil || agency.Phone != nil || agency.Email != nil {
		t.Errorf("Execute() = %#v, want every blank optional field absent", agency)
	}
}

func TestCreateAgencyPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrCUITInUse
	repository := &gatewayfake.AgencyRepository{CreateErr: wantErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", BusinessName: "Lotes del Sur",
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestCreateAgencyClassifiesUnexpectedRepositoryError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("syntax error at end of input")
	repository := &gatewayfake.AgencyRepository{CreateErr: rawErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", BusinessName: "Lotes del Sur",
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

	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", BusinessName: "Lotes del Sur",
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
	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: rawErr}
	createAgency := NewCreateAgency(repository, users)

	_, err := createAgency.Execute(context.Background(), CreateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", BusinessName: "Lotes del Sur",
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
