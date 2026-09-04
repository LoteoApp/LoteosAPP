package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestListUsersRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	listUsers := NewListUsers(repository)

	_, err := listUsers.Execute(context.Background(), []string{domain.RolAdministrativo}, false)

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.ListByRolesCalls != 0 {
		t.Error("Execute() should not query the repository when actor is not administrador")
	}
}

func TestListUsersReturnsOnlyActiveByDefault(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{
		ListByRolesResult: []domain.Usuario{{ID: "u-1", Rol: domain.RolAdministrativo}},
	}
	listUsers := NewListUsers(repository)

	usuarios, err := listUsers.Execute(context.Background(), []string{domain.RolAdministrador}, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(usuarios) != 1 || usuarios[0].ID != "u-1" {
		t.Errorf("Execute() = %#v", usuarios)
	}
	if len(repository.ListByRolesInput) != 4 {
		t.Fatalf("Execute() queried roles = %v, want the 4 gestionable roles", repository.ListByRolesInput)
	}
	for _, rol := range []domain.Rol{
		domain.RolAdministrativo, domain.RolAgrimensor, domain.RolEscribano, domain.RolInmobiliaria,
	} {
		found := false
		for _, queried := range repository.ListByRolesInput {
			if queried == rol {
				found = true
			}
		}
		if !found {
			t.Errorf("Execute() should query rol %q, got %v", rol, repository.ListByRolesInput)
		}
	}
	if repository.ListByRolesInactive {
		t.Error("Execute() should not include inactive users by default")
	}
}

func TestListUsersIncludesInactiveWhenAsked(t *testing.T) {
	t.Parallel()

	baja := time.Now()
	repository := &gatewayfake.UserRepository{
		ListByRolesResult: []domain.Usuario{
			{ID: "u-1", Rol: domain.RolEscribano},
			{ID: "u-2", Rol: domain.RolEscribano, FechaBaja: &baja},
		},
	}
	listUsers := NewListUsers(repository)

	usuarios, err := listUsers.Execute(context.Background(), []string{domain.RolAdministrador}, true)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(usuarios) != 2 {
		t.Fatalf("Execute() returned %d users, want 2", len(usuarios))
	}
	if usuarios[1].Activo() {
		t.Error("Execute() should keep the fecha de baja of an inactive user")
	}
	if !repository.ListByRolesInactive {
		t.Error("Execute() should ask the repository for the inactive users too")
	}
}

func TestListUsersWrapsRepositoryFailure(t *testing.T) {
	t.Parallel()

	cause := errors.New("connection refused")
	repository := &gatewayfake.UserRepository{ListByRolesErr: cause}
	listUsers := NewListUsers(repository)

	_, err := listUsers.Execute(context.Background(), []string{domain.RolAdministrador}, false)

	assertDatabaseUnavailable(t, err, cause)
}
