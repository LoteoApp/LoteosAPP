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

	_, err := createClient.Execute(context.Background(), CreateClientInput{
		ActorRoles: []string{domain.RolAgrimensor}, Subject: "sb-1",
		Nombre: "Ana", Apellido: "Perez", DNI: "30111222",
	})

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

			_, err := createClient.Execute(context.Background(), CreateClientInput{
				ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
				Nombre: test.nombre, Apellido: test.apellido, DNI: test.dni,
			})

			if !errors.Is(err, domain.ErrClienteInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrClienteInvalido)
			}
			if repository.CreateCalls != 0 {
				t.Error("Execute() should not call repository with invalid input")
			}
		})
	}
}

func TestCreateClientRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{}
	createClient := NewCreateClient(repository, users)

	_, err := createClient.Execute(context.Background(), CreateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		Nombre: "Ana", Apellido: "Perez", DNI: "30111222", Email: ptr("ana.example.com"),
	})

	if !errors.Is(err, domain.ErrEmailInvalido) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailInvalido)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not call repository with an invalid email")
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

			cliente, err := createClient.Execute(context.Background(), CreateClientInput{
				ActorRoles: []string{test.rol}, Subject: "sb-1",
				Nombre: " Ana ", Apellido: " Perez ", DNI: " 30111222 ",
				Celular: ptr(" 1122334455 "), Email: ptr("  ana@example.com  "),
			})
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if cliente.Nombre != "Ana" || cliente.Apellido != "Perez" || cliente.DNI != "30111222" {
				t.Errorf("Execute() cliente = %#v", cliente)
			}
			if cliente.Celular == nil || *cliente.Celular != "1122334455" {
				t.Errorf("Execute() celular = %v, want it trimmed", cliente.Celular)
			}
			if cliente.Email == nil || *cliente.Email != "ana@example.com" {
				t.Errorf("Execute() email = %v, want it trimmed", cliente.Email)
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

func TestCreateClientDropsBlankOptionalFields(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createClient := NewCreateClient(repository, users)

	cliente, err := createClient.Execute(context.Background(), CreateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		Nombre: "Ana", Apellido: "Perez", DNI: "30111222",
		Celular: ptr("   "), Email: ptr(""),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if cliente.Celular != nil || cliente.Email != nil {
		t.Errorf("Execute() celular = %v, email = %v, want both absent", cliente.Celular, cliente.Email)
	}
}

func TestCreateClientPropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrDNIEnUso
	repository := &gatewayfake.ClienteRepository{CreateErr: wantErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createClient := NewCreateClient(repository, users)

	_, err := createClient.Execute(context.Background(), CreateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		Nombre: "Ana", Apellido: "Perez", DNI: "30111222",
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

func TestCreateClientLeavesUnexpectedRepositoryErrorUnclassified(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("syntax error at end of input")
	repository := &gatewayfake.ClienteRepository{CreateErr: rawErr}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDResult: domain.Usuario{ID: "user-1"}}
	createClient := NewCreateClient(repository, users)

	_, err := createClient.Execute(context.Background(), CreateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		Nombre: "Ana", Apellido: "Perez", DNI: "30111222",
	})

	if !errors.Is(err, rawErr) {
		t.Fatalf("Execute() error = %v, want %v", err, rawErr)
	}
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		t.Errorf("Execute() error = %v, want it unclassified so it surfaces as a 500", err)
	}
}

func TestCreateClientRejectsActorWithoutUsuario(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: domain.ErrUsuarioNoEncontrado}
	createClient := NewCreateClient(repository, users)

	_, err := createClient.Execute(context.Background(), CreateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		Nombre: "Ana", Apellido: "Perez", DNI: "30111222",
	})

	if !errors.Is(err, domain.ErrActorNoAprovisionado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrActorNoAprovisionado)
	}
	if errors.Is(err, domain.ErrUsuarioNoEncontrado) {
		t.Error("Execute() should not answer a create with the not-found error of another feature")
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}

func TestCreateClientPropagatesActorResolutionError(t *testing.T) {
	t.Parallel()

	rawErr := errors.New("connection refused")
	repository := &gatewayfake.ClienteRepository{}
	users := &gatewayfake.UserRepository{FindByAuthProviderIDErr: rawErr}
	createClient := NewCreateClient(repository, users)

	_, err := createClient.Execute(context.Background(), CreateClientInput{
		ActorRoles: []string{domain.RolAdministrador}, Subject: "sb-1",
		Nombre: "Ana", Apellido: "Perez", DNI: "30111222",
	})

	if !errors.Is(err, rawErr) {
		t.Fatalf("Execute() error = %v, want %v", err, rawErr)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not call repository when actor resolution fails")
	}
}
