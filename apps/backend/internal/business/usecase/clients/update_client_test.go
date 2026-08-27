package clients

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

func TestUpdateClientRejectsUnauthorizedRole(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{}
	updateClient := NewUpdateClient(repository, users)

	_, err := updateClient.Execute(context.Background(), []string{domain.RolEscribano}, "sb-1", "client-1", ptr("Ana"), ptr("Perez"), ptr("30111222"), nil, nil)

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository when actor is not authorized")
	}
}

func TestUpdateClientRejectsBlankFields(t *testing.T) {
	tests := []struct {
		name     string
		nombre   *string
		apellido *string
		dni      *string
	}{
		{name: "blank nombre", nombre: ptr("  "), apellido: ptr("Perez"), dni: ptr("30111222")},
		{name: "blank apellido", nombre: ptr("Ana"), apellido: ptr(""), dni: ptr("30111222")},
		{name: "blank dni", nombre: ptr("Ana"), apellido: ptr("Perez"), dni: ptr("   ")},
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
				t.Error("Execute() should not call repository with an invalid field")
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

	_, err := updateClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "client-1", ptr("Ana"), ptr("Perez"), ptr("30111222"), nil, nil)

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

	cliente, err := updateClient.Execute(context.Background(), []string{domain.RolInmobiliaria}, "sb-1", "client-1", ptr("Ana"), ptr("Perez"), ptr("30111222"), nil, nil)
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

func TestUpdateClientPartialUpdateOnlyForwardsPresentFields(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateClient := NewUpdateClient(repository, users)

	celular := "1122334455"
	// Only celular is sent; nombre, apellido, dni and email are omitted
	// (nil) and must reach the repository as nil too, not as empty
	// strings that would wipe the existing values.
	_, err := updateClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "client-1", nil, nil, nil, &celular, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := repository.UpdateInput
	if got.Nombre != nil || got.Apellido != nil || got.DNI != nil || got.Email != nil {
		t.Errorf("Execute() should forward omitted fields as nil, got %#v", got)
	}
	if got.Celular == nil || *got.Celular != celular {
		t.Errorf("Execute() celular = %v, want %q", got.Celular, celular)
	}
}

func TestUpdateClientPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrClienteNoEncontrado
	repository := &gatewayfake.ClienteRepository{UpdateErr: wantErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateClient := NewUpdateClient(repository, users)

	_, err := updateClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "client-1", ptr("Ana"), ptr("Perez"), ptr("30111222"), nil, nil)

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestUpdateClientWrapsUnexpectedRepositoryError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("connection refused")
	repository := &gatewayfake.ClienteRepository{UpdateErr: rawErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateClient := NewUpdateClient(repository, users)

	_, err := updateClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "client-1", ptr("Ana"), ptr("Perez"), ptr("30111222"), nil, nil)

	if !errors.Is(err, rawErr) {
		t.Fatalf("Execute() error = %v, want it to wrap %v", err, rawErr)
	}
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) {
		t.Fatalf("Execute() error = %v, want a *domain.Error", err)
	}
	if domainErr.Kind != domain.KindUnavailable {
		t.Errorf("Execute() error kind = %q, want %q", domainErr.Kind, domain.KindUnavailable)
	}
}
