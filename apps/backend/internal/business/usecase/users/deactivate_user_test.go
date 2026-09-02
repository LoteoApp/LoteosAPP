package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestDeactivateUserRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: activeManagedUser()}
	deactivateUser := NewDeactivateUser(repository)

	err := deactivateUser.Execute(context.Background(), []string{domain.RolAdministrativo}, "admin-sub", "user-1")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not give anyone de baja when actor is not administrador")
	}
}

func TestDeactivateUserHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:                  activeManagedUser(),
		FindByAuthProviderIDResult: domain.Usuario{ID: "admin-1", Rol: domain.RolAdministrador},
	}
	deactivateUser := NewDeactivateUser(repository)

	if err := deactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.SoftDeletedID != "user-1" {
		t.Errorf("Execute() gave de baja %q, want %q", repository.SoftDeletedID, "user-1")
	}
	if repository.SoftDeletedActor != "admin-1" {
		t.Errorf("Execute() usuario_modificacion = %q, want %q", repository.SoftDeletedActor, "admin-1")
	}
}

func TestDeactivateUserRejectsUnknownID(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByIDErr: domain.ErrUsuarioNoEncontrado}
	deactivateUser := NewDeactivateUser(repository)

	err := deactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1")

	if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not give de baja a user it could not find")
	}
}

func TestDeactivateUserRejectsRolesThisABMDoesNotManage(t *testing.T) {
	t.Parallel()

	for _, rol := range []string{domain.RolAgrimensor, domain.RolAdministrador} {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.UserRepository{
				FoundByID: domain.Usuario{ID: "user-2", Rol: domain.Rol(rol)},
			}
			deactivateUser := NewDeactivateUser(repository)

			err := deactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-2")

			if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
			}
			if repository.SoftDeleteCalls != 0 {
				t.Error("Execute() should not give de baja a user of a role this ABM doesn't manage")
			}
		})
	}
}

func TestDeactivateUserRejectsAlreadyInactive(t *testing.T) {
	t.Parallel()

	baja := time.Now()
	inactive := activeManagedUser()
	inactive.FechaBaja = &baja
	repository := &gatewayfake.UserRepository{FoundByID: inactive}
	deactivateUser := NewDeactivateUser(repository)

	err := deactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1")

	if !errors.Is(err, domain.ErrUsuarioDadoDeBaja) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioDadoDeBaja)
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not repeat the baja of an inactive user")
	}
}

func TestDeactivateUserWrapsActorLookupFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:               activeManagedUser(),
		FindByAuthProviderIDErr: errors.New("connection refused"),
	}
	deactivateUser := NewDeactivateUser(repository)

	err := deactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1")

	if err == nil {
		t.Fatal("Execute() error = nil, want the actor lookup failure")
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not give de baja when the actor cannot be resolved")
	}
}

func TestDeactivateUserReportsActorNotProvisioned(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:               activeManagedUser(),
		FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado,
	}
	deactivateUser := NewDeactivateUser(repository)

	err := deactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1")

	if !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
}

func TestDeactivateUserWrapsSoftDeleteFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:     activeManagedUser(),
		SoftDeleteErr: errors.New("connection refused"),
	}
	deactivateUser := NewDeactivateUser(repository)

	err := deactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1")

	if err == nil {
		t.Fatal("Execute() error = nil, want the soft delete failure")
	}
}
