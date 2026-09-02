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
	updateUser := NewUpdateUser(repository)

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
			updateUser := NewUpdateUser(repository)

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
	updateUser := NewUpdateUser(repository)

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
	updateUser := NewUpdateUser(repository)

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
	updateUser := NewUpdateUser(repository)

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
	updateUser := NewUpdateUser(repository)

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

	repository := &gatewayfake.UserRepository{
		FoundByID:               activeManagedUser(),
		FindByAuthProviderIDErr: errors.New("connection refused"),
	}
	updateUser := NewUpdateUser(repository)

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-1", Nombre: stringPtr("Ana"),
	})

	if err == nil {
		t.Fatal("Execute() error = nil, want the actor lookup failure")
	}
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
	updateUser := NewUpdateUser(repository)

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-1", Nombre: stringPtr("Ana"),
	})

	if !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
}

func TestUpdateUserWrapsUpdateFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID: activeManagedUser(),
		UpdateErr: errors.New("connection refused"),
	}
	updateUser := NewUpdateUser(repository)

	_, err := updateUser.Execute(context.Background(), UpdateUserInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "admin-sub", ID: "user-1", Nombre: stringPtr("Ana"),
	})

	if err == nil {
		t.Fatal("Execute() error = nil, want the update failure")
	}
}
