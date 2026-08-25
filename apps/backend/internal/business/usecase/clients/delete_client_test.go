package clients

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestDeleteClientRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	tests := []string{domain.RolAdministrativo, domain.RolInmobiliaria, domain.RolAgrimensor, domain.RolEscribano}

	for _, rol := range tests {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.ClienteRepository{}
			users := &gatewayfake.UserRepository{}
			deleteClient := NewDeleteClient(repository, users)

			err := deleteClient.Execute(context.Background(), []string{rol}, "sb-1", "client-1")

			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if repository.SoftDeleteCalls != 0 {
				t.Error("Execute() should not call repository when actor is not administrador")
			}
		})
	}
}

func TestDeleteClientHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	deleteClient := NewDeleteClient(repository, users)

	err := deleteClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "client-1")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if repository.SoftDeleteCalls != 1 {
		t.Errorf("Execute() repository.SoftDelete calls = %d, want 1", repository.SoftDeleteCalls)
	}
	if repository.SoftDeletedID != "client-1" {
		t.Errorf("Execute() soft deleted id = %q, want %q", repository.SoftDeletedID, "client-1")
	}
	if repository.SoftDeletedActor != "user-1" {
		t.Errorf("Execute() soft deleted actor = %q, want %q", repository.SoftDeletedActor, "user-1")
	}
}

func TestDeleteClientPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrClienteNoEncontrado
	repository := &gatewayfake.ClienteRepository{SoftDeleteErr: wantErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	deleteClient := NewDeleteClient(repository, users)

	err := deleteClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "client-1")

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestDeleteClientPropagatesActorResolutionError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrUsuarioNoEncontrado
	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: wantErr}
	deleteClient := NewDeleteClient(repository, users)

	err := deleteClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "client-1")

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if repository.SoftDeleteCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}
