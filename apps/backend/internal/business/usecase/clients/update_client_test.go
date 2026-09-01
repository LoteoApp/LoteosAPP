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

	_, err := updateClient.Execute(context.Background(), UpdateClientInput{
		ActorRoles: []string{domain.RolEscribano}, Subject: "sb-1", ID: "client-1",
		Nombre: ptr("Ana"), Apellido: ptr("Perez"), DNI: ptr("30111222"),
	})

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

			_, err := updateClient.Execute(context.Background(), UpdateClientInput{
				ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "client-1",
				Nombre: test.nombre, Apellido: test.apellido, DNI: test.dni,
			})

			if !errors.Is(err, domain.ErrClienteInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrClienteInvalido)
			}
			if repository.UpdateCalls != 0 {
				t.Error("Execute() should not call repository with an invalid field")
			}
		})
	}
}

func TestUpdateClientRejectsRequestWithoutChanges(t *testing.T) {
	tests := []struct {
		name    string
		celular *string
		email   *string
	}{
		{name: "no fields at all"},
		{name: "only blank optional fields", celular: ptr("  "), email: ptr("")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.ClienteRepository{}
			users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
			updateClient := NewUpdateClient(repository, users)

			_, err := updateClient.Execute(context.Background(), UpdateClientInput{
				ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "client-1",
				Celular: test.celular, Email: test.email,
			})

			if !errors.Is(err, domain.ErrClienteSinCambios) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrClienteSinCambios)
			}
			if repository.UpdateCalls != 0 {
				t.Error("Execute() should not touch the audit columns when nothing changes")
			}
		})
	}
}

func TestUpdateClientRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{}
	updateClient := NewUpdateClient(repository, users)

	_, err := updateClient.Execute(context.Background(), UpdateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "client-1",
		Email: ptr("ana.example.com"),
	})

	if !errors.Is(err, domain.ErrEmailInvalido) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailInvalido)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository with an invalid email")
	}
}

func TestUpdateClientRejectsActorWithoutUsuario(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado}
	updateClient := NewUpdateClient(repository, users)

	_, err := updateClient.Execute(context.Background(), UpdateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "client-1",
		Nombre: ptr("Ana"), Apellido: ptr("Perez"), DNI: ptr("30111222"),
	})

	if !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
	if repository.UpdateCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}

func TestUpdateClientPropagatesActorResolutionError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("connection refused")
	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: rawErr}
	updateClient := NewUpdateClient(repository, users)

	_, err := updateClient.Execute(context.Background(), UpdateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "client-1",
		Nombre: ptr("Ana"), Apellido: ptr("Perez"), DNI: ptr("30111222"),
	})

	if !errors.Is(err, rawErr) {
		t.Fatalf("Execute() error = %v, want %v", err, rawErr)
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

	cliente, err := updateClient.Execute(context.Background(), UpdateClientInput{
		ActorRoles: []string{domain.RolInmobiliaria}, Subject: "sb-1", ID: "client-1",
		Nombre: ptr("Ana"), Apellido: ptr("Perez"), DNI: ptr("30111222"),
	})
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

	// Only celular is sent; nombre, apellido, dni and email are omitted
	// (nil) and must reach the repository as nil too, not as empty
	// strings that would wipe the existing values.
	_, err := updateClient.Execute(context.Background(), UpdateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "client-1",
		Celular: ptr(" 1122334455 "),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := repository.UpdateInput
	if got.Nombre != nil || got.Apellido != nil || got.DNI != nil || got.Email != nil {
		t.Errorf("Execute() should forward omitted fields as nil, got %#v", got)
	}
	if got.Celular == nil || *got.Celular != "1122334455" {
		t.Errorf("Execute() celular = %v, want it trimmed", got.Celular)
	}
}

func TestUpdateClientPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrClienteNoEncontrado
	repository := &gatewayfake.ClienteRepository{UpdateErr: wantErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateClient := NewUpdateClient(repository, users)

	_, err := updateClient.Execute(context.Background(), UpdateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "client-1",
		Nombre: ptr("Ana"), Apellido: ptr("Perez"), DNI: ptr("30111222"),
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestUpdateClientLeavesUnexpectedRepositoryErrorUnclassified(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("syntax error at end of input")
	repository := &gatewayfake.ClienteRepository{UpdateErr: rawErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	updateClient := NewUpdateClient(repository, users)

	_, err := updateClient.Execute(context.Background(), UpdateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1", ID: "client-1",
		Nombre: ptr("Ana"), Apellido: ptr("Perez"), DNI: ptr("30111222"),
	})

	if !errors.Is(err, rawErr) {
		t.Fatalf("Execute() error = %v, want %v", err, rawErr)
	}
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		t.Errorf("Execute() error = %v, want it unclassified so it surfaces as a 500", err)
	}
}
