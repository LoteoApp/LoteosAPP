package agencies

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestUpdateAgencyRejectsUnauthorizedRole(t *testing.T) {
	t.Parallel()

	roles := []string{domain.RolAdministrativo, domain.RolAgrimensor, domain.RolEscribano, domain.RolInmobiliaria}

	for _, rol := range roles {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.AgencyRepository{}
			users := &gatewayfake.UserRepository{}
			updateAgency := NewUpdateAgency(repository, users)

			_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
				ActorRoles: []string{rol}, Subject: "sb-1", ID: "agency-1", BusinessName: ptr("Lotes del Sur"),
			})

			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if repository.UpdateCalls != 0 {
				t.Error("Execute() should not call repository when actor is not authorized")
			}
		})
	}
}

func TestUpdateAgencyRejectsBlankRazonSocial(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", BusinessName: ptr("   "),
	})

	if !errors.Is(err, domain.ErrInvalidAgency) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidAgency)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository with a blank razon social")
	}
}

func TestUpdateAgencyRejectsInvalidCUIT(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", CUIT: ptr("123"),
	})

	if !errors.Is(err, domain.ErrInvalidCUIT) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInvalidCUIT)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository with an invalid cuit")
	}
}

func TestUpdateAgencyRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", Email: ptr("contacto.example.com"),
	})

	if !errors.Is(err, domain.ErrEmailInvalido) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailInvalido)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository with an invalid email")
	}
}

func TestUpdateAgencyRejectsAnEmptyChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input UpdateAgencyInput
	}{
		{
			name:  "no field at all",
			input: UpdateAgencyInput{ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1"},
		},
		{
			name: "only blank optional fields",
			input: UpdateAgencyInput{
				ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1",
				CUIT: ptr("  "), Phone: ptr(""), Email: ptr("   "),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.AgencyRepository{}
			users := &gatewayfake.UserRepository{}
			updateAgency := NewUpdateAgency(repository, users)

			_, err := updateAgency.Execute(context.Background(), test.input)

			if !errors.Is(err, domain.ErrEmptyAgencyUpdate) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmptyAgencyUpdate)
			}
			if repository.UpdateCalls != 0 {
				t.Error("Execute() should not stamp a modification for a change that isn't one")
			}
		})
	}
}

func TestUpdateAgencyHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateAgency := NewUpdateAgency(repository, users)

	updated, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1",
		BusinessName: ptr("  Lotes del Sur SRL  "), CUIT: ptr("30-71234567-8"),
		Phone: ptr(" 3415551234 "), Email: ptr(" contacto@lotesdelsur.com "),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if updated.BusinessName != "Lotes del Sur SRL" {
		t.Errorf("Execute() razon social = %q, want it trimmed", updated.BusinessName)
	}
	if repository.UpdateInput.CUIT == nil || *repository.UpdateInput.CUIT != "30712345678" {
		t.Errorf("Execute() cuit = %v, want it normalized to digits", repository.UpdateInput.CUIT)
	}
	if repository.UpdateInput.ModifiedBy != "user-1" {
		t.Errorf("Execute() usuario modificacion = %q, want %q", repository.UpdateInput.ModifiedBy, "user-1")
	}
	if repository.UpdateInput.ID != "agency-1" {
		t.Errorf("Execute() id = %q, want %q", repository.UpdateInput.ID, "agency-1")
	}
}

func TestUpdateAgencyLeavesOmittedFieldsUnchanged(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateAgency := NewUpdateAgency(repository, users)

	if _, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1",
		Phone: ptr("3415551234"),
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.UpdateInput.BusinessName != nil {
		t.Errorf("Execute() razon social = %v, want it absent", repository.UpdateInput.BusinessName)
	}
	if repository.UpdateInput.CUIT != nil || repository.UpdateInput.Email != nil {
		t.Errorf("Execute() update = %#v, want the omitted fields absent", repository.UpdateInput)
	}
}

func TestUpdateAgencyPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrAgencyNotFound
	repository := &gatewayfake.AgencyRepository{UpdateErr: wantErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", BusinessName: ptr("Lotes del Sur"),
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestUpdateAgencyClassifiesUnexpectedRepositoryError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("syntax error at end of input")
	repository := &gatewayfake.AgencyRepository{UpdateErr: rawErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", BusinessName: ptr("Lotes del Sur"),
	})

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
	if !errors.Is(err, rawErr) {
		t.Errorf("Execute() error = %v, want it to carry %v as Cause for the log", err, rawErr)
	}
}

func TestUpdateAgencyRejectsActorWithoutUsuario(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", BusinessName: ptr("Lotes del Sur"),
	})

	if !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}

func TestUpdateAgencyClassifiesActorResolutionError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("connection refused")
	repository := &gatewayfake.AgencyRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: rawErr}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", BusinessName: ptr("Lotes del Sur"),
	})

	if !errors.Is(err, domain.ErrDatabaseUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrDatabaseUnavailable)
	}
	if !errors.Is(err, rawErr) {
		t.Errorf("Execute() error = %v, want it to carry %v as Cause for the log", err, rawErr)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}
