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

			repository := &gatewayfake.InmobiliariaRepository{}
			users := &gatewayfake.UserRepository{}
			updateAgency := NewUpdateAgency(repository, users)

			_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
				ActorRoles: []string{rol}, Subject: "sb-1", ID: "agency-1", RazonSocial: ptr("Lotes del Sur"),
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

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", RazonSocial: ptr("   "),
	})

	if !errors.Is(err, domain.ErrInmobiliariaInvalida) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInmobiliariaInvalida)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository with a blank razon social")
	}
}

func TestUpdateAgencyRejectsInvalidCUIT(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", CUIT: ptr("123"),
	})

	if !errors.Is(err, domain.ErrCUITInvalido) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrCUITInvalido)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository with an invalid cuit")
	}
}

func TestUpdateAgencyRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
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
				CUIT: ptr("  "), Telefono: ptr(""), Email: ptr("   "),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.InmobiliariaRepository{}
			users := &gatewayfake.UserRepository{}
			updateAgency := NewUpdateAgency(repository, users)

			_, err := updateAgency.Execute(context.Background(), test.input)

			if !errors.Is(err, domain.ErrInmobiliariaSinCambios) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInmobiliariaSinCambios)
			}
			if repository.UpdateCalls != 0 {
				t.Error("Execute() should not stamp a modification for a change that isn't one")
			}
		})
	}
}

func TestUpdateAgencyHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateAgency := NewUpdateAgency(repository, users)

	updated, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1",
		RazonSocial: ptr("  Lotes del Sur SRL  "), CUIT: ptr("30-71234567-8"),
		Telefono: ptr(" 3415551234 "), Email: ptr(" contacto@lotesdelsur.com "),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if updated.RazonSocial != "Lotes del Sur SRL" {
		t.Errorf("Execute() razon social = %q, want it trimmed", updated.RazonSocial)
	}
	if repository.UpdateInput.CUIT == nil || *repository.UpdateInput.CUIT != "30712345678" {
		t.Errorf("Execute() cuit = %v, want it normalized to digits", repository.UpdateInput.CUIT)
	}
	if repository.UpdateInput.UsuarioModificacion != "user-1" {
		t.Errorf("Execute() usuario modificacion = %q, want %q", repository.UpdateInput.UsuarioModificacion, "user-1")
	}
	if repository.UpdateInput.ID != "agency-1" {
		t.Errorf("Execute() id = %q, want %q", repository.UpdateInput.ID, "agency-1")
	}
}

func TestUpdateAgencyLeavesOmittedFieldsUnchanged(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateAgency := NewUpdateAgency(repository, users)

	if _, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1",
		Telefono: ptr("3415551234"),
	}); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.UpdateInput.RazonSocial != nil {
		t.Errorf("Execute() razon social = %v, want it absent", repository.UpdateInput.RazonSocial)
	}
	if repository.UpdateInput.CUIT != nil || repository.UpdateInput.Email != nil {
		t.Errorf("Execute() update = %#v, want the omitted fields absent", repository.UpdateInput)
	}
}

func TestUpdateAgencyPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrInmobiliariaNoEncontrada
	repository := &gatewayfake.InmobiliariaRepository{UpdateErr: wantErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", RazonSocial: ptr("Lotes del Sur"),
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestUpdateAgencyLeavesUnexpectedRepositoryErrorUnclassified(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("syntax error at end of input")
	repository := &gatewayfake.InmobiliariaRepository{UpdateErr: rawErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", RazonSocial: ptr("Lotes del Sur"),
	})

	if !errors.Is(err, rawErr) {
		t.Fatalf("Execute() error = %v, want %v", err, rawErr)
	}
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		t.Errorf("Execute() error = %v, want it unclassified so it surfaces as a 500", err)
	}
}

func TestUpdateAgencyRejectsActorWithoutUsuario(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", RazonSocial: ptr("Lotes del Sur"),
	})

	if !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}

func TestUpdateAgencyPropagatesActorResolutionError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("connection refused")
	repository := &gatewayfake.InmobiliariaRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: rawErr}
	updateAgency := NewUpdateAgency(repository, users)

	_, err := updateAgency.Execute(context.Background(), UpdateAgencyInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "agency-1", RazonSocial: ptr("Lotes del Sur"),
	})

	if !errors.Is(err, rawErr) {
		t.Fatalf("Execute() error = %v, want %v", err, rawErr)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}
