package users

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func stringPtr(value string) *string {
	return &value
}

func activeManagedUser() domain.Usuario {
	return domain.Usuario{ID: "user-1", Rol: domain.RolEscribano, Nombre: "Ana", Apellido: "Gómez"}
}

func TestUpdateUserRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: activeManagedUser()}
	updateUser := NewUpdateUser(repository, &gatewayfake.AgencyRepository{})

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrativo}, Subject: "admin-sub", ID: "user-1", Nombre: stringPtr("Ana María"),
	})

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not update when actor is not administrador")
	}
}

func TestUpdateUserRejectsBlankProfileFields(t *testing.T) {
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

			repository := &gatewayfake.UserRepository{FoundByID: activeManagedUser()}
			updateUser := NewUpdateUser(repository, &gatewayfake.AgencyRepository{})

			_, err := updateUser.Execute(context.Background(), UpdateUserInput{
				ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-1",
				Nombre: test.nombre, Apellido: test.apellido,
			})

			if !errors.Is(err, domain.ErrPerfilInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrPerfilInvalido)
			}
			if repository.UpdateCalls != 0 {
				t.Error("Execute() should not update with a blank profile field")
			}
		})
	}
}

func TestUpdateUserRejectsEmptyUpdate(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: activeManagedUser()}
	updateUser := NewUpdateUser(repository, &gatewayfake.AgencyRepository{})

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-1",
	})

	if !errors.Is(err, domain.ErrUsuarioSinCambios) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioSinCambios)
	}
	if repository.FindByIDCalls != 0 || repository.UpdateCalls != 0 {
		t.Error("Execute() should not look up or update a request with no fields")
	}
}

func TestUpdateUserTrimsAndPersistsOnlyTheFieldsSent(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:                  activeManagedUser(),
		FindByAuthProviderIDResult: domain.Usuario{ID: "admin-1", Rol: domain.RolAdministrador},
	}
	updateUser := NewUpdateUser(repository, &gatewayfake.AgencyRepository{})

	updated, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-1",
		Nombre: stringPtr("  Ana María  "),
	})
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

func TestUpdateUserRejectsUnknownID(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByIDErr: domain.ErrUsuarioNoEncontrado}
	updateUser := NewUpdateUser(repository, &gatewayfake.AgencyRepository{})

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-1", Nombre: stringPtr("Ana"),
	})

	if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not update a user it could not find")
	}
}

func TestUpdateUserRejectsRolesThisABMDoesNotManage(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID: domain.Usuario{ID: "user-2", Rol: domain.RolAdministrador},
	}
	updateUser := NewUpdateUser(repository, &gatewayfake.AgencyRepository{})

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-2", Nombre: stringPtr("Ana"),
	})

	if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not update a user of a role this ABM doesn't manage")
	}
}

func TestUpdateUserWrapsActorLookupFailure(t *testing.T) {
	t.Parallel()

	cause := errors.New("connection refused")
	repository := &gatewayfake.UserRepository{
		FoundByID:               activeManagedUser(),
		FindByAuthProviderIDErr: cause,
	}
	updateUser := NewUpdateUser(repository, &gatewayfake.AgencyRepository{})

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-1", Nombre: stringPtr("Ana"),
	})

	assertDatabaseUnavailable(t, err, cause)
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not update when the actor cannot be resolved")
	}
}

func TestUpdateUserReportsActorNotProvisioned(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:               activeManagedUser(),
		FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado,
	}
	updateUser := NewUpdateUser(repository, &gatewayfake.AgencyRepository{})

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-1", Nombre: stringPtr("Ana"),
	})

	if !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
}

func TestUpdateUserWrapsUpdateFailure(t *testing.T) {
	t.Parallel()

	cause := errors.New("connection refused")
	repository := &gatewayfake.UserRepository{
		FoundByID: activeManagedUser(),
		UpdateErr: cause,
	}
	updateUser := NewUpdateUser(repository, &gatewayfake.AgencyRepository{})

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-1", Nombre: stringPtr("Ana"),
	})

	assertDatabaseUnavailable(t, err, cause)
}

func inmobiliariaWithoutAgency() domain.Usuario {
	return domain.Usuario{ID: "user-3", Rol: domain.RolInmobiliaria, Nombre: "Luis", Apellido: "Pérez"}
}

func TestUpdateUserAssignsAgencyToInmobiliariaWithoutOne(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:                  inmobiliariaWithoutAgency(),
		FindByAuthProviderIDResult: domain.Usuario{ID: "admin-1", Rol: domain.RolAdministrador},
	}
	agencies := &gatewayfake.AgencyRepository{FoundByID: domain.Agency{ID: "agency-1"}}
	updateUser := NewUpdateUser(repository, agencies)

	updated, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-3",
		AgencyID: stringPtr("  agency-1  "),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if agencies.FindByIDInput != "agency-1" {
		t.Errorf("Execute() looked up agency %q, want %q", agencies.FindByIDInput, "agency-1")
	}
	if updated.AgencyID == nil || *updated.AgencyID != "agency-1" {
		t.Errorf("Execute() agency = %v, want %q", updated.AgencyID, "agency-1")
	}
	if repository.UpdateInput.Nombre != nil || repository.UpdateInput.Apellido != nil {
		t.Error("Execute() should leave nombre and apellido unchanged when only the agency is sent")
	}
}

func TestUpdateUserRejectsBlankAgency(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: inmobiliariaWithoutAgency()}
	agencies := &gatewayfake.AgencyRepository{}
	updateUser := NewUpdateUser(repository, agencies)

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-3", AgencyID: stringPtr("   "),
	})

	if !errors.Is(err, domain.ErrAgenciaRequerida) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgenciaRequerida)
	}
	if repository.FindByIDCalls != 0 || agencies.FindByIDCalls != 0 || repository.UpdateCalls != 0 {
		t.Error("Execute() should not look up or update with a blank agency")
	}
}

func TestUpdateUserRejectsAgencyForOtherRoles(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: activeManagedUser()}
	agencies := &gatewayfake.AgencyRepository{}
	updateUser := NewUpdateUser(repository, agencies)

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-1", AgencyID: stringPtr("agency-1"),
	})

	if !errors.Is(err, domain.ErrAgenciaNoAplica) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgenciaNoAplica)
	}
	if agencies.FindByIDCalls != 0 || repository.UpdateCalls != 0 {
		t.Error("Execute() should not look up the agency or update a user whose role has none")
	}
}

func TestUpdateUserRejectsReassigningAgency(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		agencyID string
	}{
		{name: "otra inmobiliaria", agencyID: "agency-2"},
		{name: "la misma inmobiliaria", agencyID: "agency-1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			target := inmobiliariaWithoutAgency()
			target.AgencyID = stringPtr("agency-1")
			repository := &gatewayfake.UserRepository{FoundByID: target}
			agencies := &gatewayfake.AgencyRepository{}
			updateUser := NewUpdateUser(repository, agencies)

			_, err := updateUser.Execute(context.Background(), UpdateUserInput{
				ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-3",
				AgencyID: stringPtr(test.agencyID),
			})

			if !errors.Is(err, domain.ErrAgenciaYaAsignada) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgenciaYaAsignada)
			}
			if agencies.FindByIDCalls != 0 || repository.UpdateCalls != 0 {
				t.Error("Execute() should not update a user that already has an agency")
			}
		})
	}
}

func TestUpdateUserPropagatesAgencyNotFound(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: inmobiliariaWithoutAgency()}
	agencies := &gatewayfake.AgencyRepository{FindByIDErr: domain.ErrAgencyNotFound}
	updateUser := NewUpdateUser(repository, agencies)

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-3", AgencyID: stringPtr("agency-9"),
	})

	if !errors.Is(err, domain.ErrAgencyNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgencyNotFound)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not update with an agency that doesn't exist or is inactive")
	}
}

func TestUpdateUserWrapsAgencyLookupFailure(t *testing.T) {
	t.Parallel()

	cause := errors.New("connection refused")
	repository := &gatewayfake.UserRepository{FoundByID: inmobiliariaWithoutAgency()}
	updateUser := NewUpdateUser(repository, &gatewayfake.AgencyRepository{FindByIDErr: cause})

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-3", AgencyID: stringPtr("agency-1"),
	})

	assertDatabaseUnavailable(t, err, cause)
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not update when the agency lookup fails")
	}
}
