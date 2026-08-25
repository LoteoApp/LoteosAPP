package clients

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestUpdateClientRejectsUnauthorizedRole(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{}
	updateClient := NewUpdateClient(repository, users)

	_, err := updateClient.Execute(context.Background(), []string{domain.RolEscribano}, "sb-1", "client-1", "Ana", "Perez", "30111222", nil, nil)

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository when actor is not authorized")
	}
}

func TestUpdateClientRejectsEmptyFields(t *testing.T) {
	tests := []struct {
		name     string
		nombre   string
		apellido string
		dni      string
	}{
		{name: "empty nombre", nombre: "  ", apellido: "Perez", dni: "30111222"},
		{name: "empty apellido", nombre: "Ana", apellido: "", dni: "30111222"},
		{name: "empty dni", nombre: "Ana", apellido: "Perez", dni: "   "},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.ClienteRepository{}
			users := &gatewayfake.UserRepository{}
			updateClient := NewUpdateClient(repository, users)

			_, err := updateClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "client-1", test.nombre, test.apellido, test.dni, nil, nil)

			if !errors.Is(err, domain.ErrClienteInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrClienteInvalido)
			}
			if repository.UpdateCalls != 0 {
				t.Error("Execute() should not call repository with invalid input")
			}
		})
	}
}

func TestUpdateClientPropagatesActorResolutionError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrUsuarioNoEncontrado
	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: wantErr}
	updateClient := NewUpdateClient(repository, users)

	_, err := updateClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "client-1", "Ana", "Perez", "30111222", nil, nil)

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}

func TestUpdateClientHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateClient := NewUpdateClient(repository, users)

	cliente, err := updateClient.Execute(context.Background(), []string{domain.RolInmobiliaria}, "sb-1", "client-1", "Ana", "Perez", "30111222", nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if cliente.ID != "client-1" {
		t.Errorf("Execute() cliente id = %q, want %q", cliente.ID, "client-1")
	}
	if cliente.UsuarioModificacion != "user-1" {
		t.Errorf("Execute() usuario modificacion = %q, want %q", cliente.UsuarioModificacion, "user-1")
	}
	if repository.UpdateCalls != 1 {
		t.Errorf("Execute() repository.Update calls = %d, want 1", repository.UpdateCalls)
	}
}

func TestUpdateClientPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrClienteNoEncontrado
	repository := &gatewayfake.ClienteRepository{UpdateErr: wantErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateClient := NewUpdateClient(repository, users)

	_, err := updateClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "client-1", "Ana", "Perez", "30111222", nil, nil)

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}
