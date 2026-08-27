package clients

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestCreateClientRejectsUnauthorizedRole(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{}
	createClient := NewCreateClient(repository, users)

	_, err := createClient.Execute(context.Background(), []string{domain.RolAgrimensor}, "sb-1", "Ana", "Perez", "30111222", nil, nil)

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not call repository when actor is not authorized")
	}
}

func TestCreateClientRejectsEmptyFields(t *testing.T) {
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
			createClient := NewCreateClient(repository, users)

			_, err := createClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", test.nombre, test.apellido, test.dni, nil, nil)

			if !errors.Is(err, domain.ErrClienteInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrClienteInvalido)
			}
			if repository.CreateCalls != 0 {
				t.Error("Execute() should not call repository with invalid input")
			}
		})
	}
}

func TestCreateClientHappyPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rol  string
	}{
		{name: "administrador", rol: domain.RolAdministrador},
		{name: "administrativo", rol: domain.RolAdministrativo},
		{name: "inmobiliaria", rol: domain.RolInmobiliaria},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.ClienteRepository{}
			users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
			createClient := NewCreateClient(repository, users)

			celular := "1122334455"
			email := "ana@example.com"
			cliente, err := createClient.Execute(context.Background(), []string{test.rol}, "sb-1", " Ana ", " Perez ", " 30111222 ", &celular, &email)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if cliente.Nombre != "Ana" || cliente.Apellido != "Perez" || cliente.DNI != "30111222" {
				t.Errorf("Execute() cliente = %#v", cliente)
			}
			if cliente.UsuarioModificacion != "user-1" {
				t.Errorf("Execute() usuario modificacion = %q, want %q", cliente.UsuarioModificacion, "user-1")
			}
			if repository.CreateCalls != 1 {
				t.Errorf("Execute() repository.Create calls = %d, want 1", repository.CreateCalls)
			}
			if users.FindByAuthProviderIDSubject != "sb-1" {
				t.Errorf("Execute() resolved subject = %q, want %q", users.FindByAuthProviderIDSubject, "sb-1")
			}
		})
	}
}

func TestCreateClientPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrDNIEnUso
	repository := &gatewayfake.ClienteRepository{CreateErr: wantErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createClient := NewCreateClient(repository, users)

	_, err := createClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "Ana", "Perez", "30111222", nil, nil)

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestCreateClientWrapsUnexpectedRepositoryError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("connection refused")
	repository := &gatewayfake.ClienteRepository{CreateErr: rawErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createClient := NewCreateClient(repository, users)

	_, err := createClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "Ana", "Perez", "30111222", nil, nil)

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

func TestCreateClientPropagatesActorResolutionError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrUsuarioNoEncontrado
	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: wantErr}
	createClient := NewCreateClient(repository, users)

	_, err := createClient.Execute(context.Background(), []string{domain.RolAdministrador}, "sb-1", "Ana", "Perez", "30111222", nil, nil)

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}
