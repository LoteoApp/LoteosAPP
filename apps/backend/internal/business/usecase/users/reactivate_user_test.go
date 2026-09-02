package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func inactiveManagedUser() domain.Usuario {
	baja := time.Now()
	inactive := activeManagedUser()
	inactive.FechaBaja = &baja
	return inactive
}

func TestReactivateUserRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: inactiveManagedUser()}
	reactivateUser := NewReactivateUser(repository)

	_, err := reactivateUser.Execute(context.Background(), []string{domain.RolAdministrativo}, "admin-sub", "user-1")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.ReactivateCalls != 0 {
		t.Error("Execute() should not reactivate anyone when actor is not administrador")
	}
}

func TestReactivateUserHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:                  inactiveManagedUser(),
		FindByAuthProviderIDResult: domain.Usuario{ID: "admin-1", Rol: domain.RolAdministrador},
	}
	reactivateUser := NewReactivateUser(repository)

	usuario, err := reactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.ReactivatedID != "user-1" {
		t.Errorf("Execute() reactivated %q, want %q", repository.ReactivatedID, "user-1")
	}
	if repository.ReactivatedActor != "admin-1" {
		t.Errorf("Execute() usuario_modificacion = %q, want %q", repository.ReactivatedActor, "admin-1")
	}
	if usuario.FechaBaja != nil {
		t.Error("Execute() should return the user with fecha_baja cleared")
	}
	if !usuario.Activo() {
		t.Error("Execute() should return an active user")
	}
}

func TestReactivateUserRejectsUnknownID(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByIDErr: domain.ErrUsuarioNoEncontrado}
	reactivateUser := NewReactivateUser(repository)

	_, err := reactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1")

	if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
	}
	if repository.ReactivateCalls != 0 {
		t.Error("Execute() should not reactivate a user it could not find")
	}
}

func TestReactivateUserRejectsRolesThisABMDoesNotManage(t *testing.T) {
	t.Parallel()

	for _, rol := range []string{domain.RolAgrimensor, domain.RolAdministrador} {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			baja := time.Now()
			repository := &gatewayfake.UserRepository{
				FoundByID: domain.Usuario{ID: "user-2", Rol: domain.Rol(rol), FechaBaja: &baja},
			}
			reactivateUser := NewReactivateUser(repository)

			_, err := reactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-2")

			if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
			}
			if repository.ReactivateCalls != 0 {
				t.Error("Execute() should not reactivate a user of a role this ABM doesn't manage")
			}
		})
	}
}

func TestReactivateUserRejectsAlreadyActive(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: activeManagedUser()}
	reactivateUser := NewReactivateUser(repository)

	_, err := reactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1")

	if !errors.Is(err, domain.ErrUsuarioYaActivo) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioYaActivo)
	}
	if repository.ReactivateCalls != 0 {
		t.Error("Execute() should not repeat the reactivation of an active user")
	}
}

func TestReactivateUserWrapsActorLookupFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:               inactiveManagedUser(),
		FindByAuthProviderIDErr: errors.New("connection refused"),
	}
	reactivateUser := NewReactivateUser(repository)

	_, err := reactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1")

	if err == nil {
		t.Fatal("Execute() error = nil, want the actor lookup failure")
	}
	if repository.ReactivateCalls != 0 {
		t.Error("Execute() should not reactivate when the actor cannot be resolved")
	}
}

func TestReactivateUserReportsActorNotProvisioned(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:               inactiveManagedUser(),
		FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado,
	}
	reactivateUser := NewReactivateUser(repository)

	_, err := reactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1")

	if !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
}

func TestReactivateUserWrapsReactivateFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		FoundByID:     inactiveManagedUser(),
		ReactivateErr: errors.New("connection refused"),
	}
	reactivateUser := NewReactivateUser(repository)

	_, err := reactivateUser.Execute(context.Background(), []string{domain.RolAdministrador}, "admin-sub", "user-1")

	if err == nil {
		t.Fatal("Execute() error = nil, want the reactivate failure")
	}
}
