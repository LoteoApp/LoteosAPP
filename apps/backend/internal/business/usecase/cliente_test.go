package usecase

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
)

type clientRepositoryFake struct {
	createCalls int
	createErr   error
	createdBy   string
	created     domain.Cliente

	listCalls  int
	listErr    error
	listBuscar string
	listResult []domain.Cliente

	updateCalls int
	updateErr   error
	updatedBy   string
	updated     domain.Cliente

	deleteCalls int
	deleteErr   error
	deletedID   string
	deletedBy   string
}

func (fake *clientRepositoryFake) Create(_ context.Context, cliente domain.Cliente, usuarioModificacion string) (domain.Cliente, error) {
	fake.createCalls++
	fake.createdBy = usuarioModificacion
	fake.created = cliente
	if fake.createErr != nil {
		return domain.Cliente{}, fake.createErr
	}
	return cliente, nil
}

func (fake *clientRepositoryFake) List(_ context.Context, buscar string) ([]domain.Cliente, error) {
	fake.listCalls++
	fake.listBuscar = buscar
	if fake.listErr != nil {
		return nil, fake.listErr
	}
	return fake.listResult, nil
}

func (fake *clientRepositoryFake) Update(_ context.Context, cliente domain.Cliente, usuarioModificacion string) (domain.Cliente, error) {
	fake.updateCalls++
	fake.updatedBy = usuarioModificacion
	fake.updated = cliente
	if fake.updateErr != nil {
		return domain.Cliente{}, fake.updateErr
	}
	return cliente, nil
}

func (fake *clientRepositoryFake) SoftDelete(_ context.Context, id, usuarioModificacion string) error {
	fake.deleteCalls++
	fake.deletedID = id
	fake.deletedBy = usuarioModificacion
	return fake.deleteErr
}

type actorRepositoryFake struct {
	userRepositoryFake

	usuarioID string
	findErr   error
	gotAuthID string
}

func (fake *actorRepositoryFake) FindByAuthProviderID(_ context.Context, authProviderID string) (domain.Usuario, error) {
	fake.gotAuthID = authProviderID
	if fake.findErr != nil {
		return domain.Usuario{}, fake.findErr
	}
	return domain.Usuario{ID: fake.usuarioID, AuthProviderID: authProviderID}, nil
}

func validCliente() domain.Cliente {
	return domain.Cliente{Nombre: "Ana", Apellido: "Pérez", DNI: "30111222"}
}

func TestClientServiceCreateAllowsManagementRoles(t *testing.T) {
	t.Parallel()

	roles := []string{domain.RolAdministrador, domain.RolAdministrativo, domain.RolInmobiliaria}

	for _, rol := range roles {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			clientes := &clientRepositoryFake{}
			usuarios := &actorRepositoryFake{usuarioID: "user-1"}
			service := NewClientService(clientes, usuarios)

			cliente, err := service.Create(context.Background(), []string{rol}, "sb-123", validCliente())
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			if cliente.Nombre != "Ana" {
				t.Errorf("Create() cliente = %#v", cliente)
			}
			if clientes.createCalls != 1 {
				t.Errorf("Create() repository calls = %d, want 1", clientes.createCalls)
			}
			if clientes.createdBy != "user-1" {
				t.Errorf("Create() usuario_modificacion = %q, want %q", clientes.createdBy, "user-1")
			}
			if usuarios.gotAuthID != "sb-123" {
				t.Errorf("Create() looked up auth provider id %q, want %q", usuarios.gotAuthID, "sb-123")
			}
		})
	}
}

func TestClientServiceCreateRejectsRolesWithoutPermission(t *testing.T) {
	t.Parallel()

	roles := []string{domain.RolAgrimensor, domain.RolEscribano}

	for _, rol := range roles {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			clientes := &clientRepositoryFake{}
			service := NewClientService(clientes, &actorRepositoryFake{})

			_, err := service.Create(context.Background(), []string{rol}, "sb-123", validCliente())

			if !errors.Is(err, domain.ErrNoAutorizado) {
				t.Fatalf("Create() error = %v, want %v", err, domain.ErrNoAutorizado)
			}
			if clientes.createCalls != 0 {
				t.Error("Create() should not reach the repository without permission")
			}
		})
	}
}

func TestClientServiceCreateRejectsIncompleteCliente(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cliente domain.Cliente
	}{
		{name: "empty nombre", cliente: domain.Cliente{Nombre: "  ", Apellido: "Pérez", DNI: "30111222"}},
		{name: "empty apellido", cliente: domain.Cliente{Nombre: "Ana", DNI: "30111222"}},
		{name: "empty dni", cliente: domain.Cliente{Nombre: "Ana", Apellido: "Pérez", DNI: " "}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			clientes := &clientRepositoryFake{}
			service := NewClientService(clientes, &actorRepositoryFake{usuarioID: "user-1"})

			_, err := service.Create(context.Background(), []string{domain.RolAdministrativo}, "sb-123", test.cliente)

			if !errors.Is(err, domain.ErrClienteInvalido) {
				t.Fatalf("Create() error = %v, want %v", err, domain.ErrClienteInvalido)
			}
			if clientes.createCalls != 0 {
				t.Error("Create() should not reach the repository with invalid input")
			}
		})
	}
}

func TestClientServiceCreateTrimsInput(t *testing.T) {
	t.Parallel()

	clientes := &clientRepositoryFake{}
	service := NewClientService(clientes, &actorRepositoryFake{usuarioID: "user-1"})

	_, err := service.Create(context.Background(), []string{domain.RolAdministrativo}, "sb-123", domain.Cliente{
		Nombre: "  Ana  ", Apellido: " Pérez ", DNI: " 30111222 ", Celular: " 1122334455 ", Email: " ana@example.com ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got := clientes.created
	if got.Nombre != "Ana" || got.Apellido != "Pérez" || got.DNI != "30111222" {
		t.Errorf("Create() persisted %#v, want trimmed values", got)
	}
	if got.Celular != "1122334455" || got.Email != "ana@example.com" {
		t.Errorf("Create() persisted contact %#v, want trimmed values", got)
	}
}

func TestClientServiceCreatePropagatesDuplicateDNI(t *testing.T) {
	t.Parallel()

	clientes := &clientRepositoryFake{createErr: domain.ErrDNIEnUso}
	service := NewClientService(clientes, &actorRepositoryFake{usuarioID: "user-1"})

	_, err := service.Create(context.Background(), []string{domain.RolAdministrativo}, "sb-123", validCliente())

	if !errors.Is(err, domain.ErrDNIEnUso) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrDNIEnUso)
	}
}

func TestClientServiceCreateFailsWhenActorIsUnknown(t *testing.T) {
	t.Parallel()

	clientes := &clientRepositoryFake{}
	usuarios := &actorRepositoryFake{findErr: domain.ErrUsuarioNoEncontrado}
	service := NewClientService(clientes, usuarios)

	_, err := service.Create(context.Background(), []string{domain.RolAdministrativo}, "sb-123", validCliente())

	if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
		t.Fatalf("Create() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
	}
	if clientes.createCalls != 0 {
		t.Error("Create() should not persist when the actor cannot be resolved")
	}
}

func TestClientServiceList(t *testing.T) {
	t.Parallel()

	t.Run("returns the repository results", func(t *testing.T) {
		t.Parallel()

		want := []domain.Cliente{{ID: "c-1", Nombre: "Ana"}}
		clientes := &clientRepositoryFake{listResult: want}
		service := NewClientService(clientes, &actorRepositoryFake{})

		got, err := service.List(context.Background(), []string{domain.RolInmobiliaria}, "  ana  ")
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(got) != 1 || got[0].ID != "c-1" {
			t.Errorf("List() = %#v", got)
		}
		if clientes.listBuscar != "ana" {
			t.Errorf("List() search term = %q, want %q", clientes.listBuscar, "ana")
		}
	})

	t.Run("rejects roles without permission", func(t *testing.T) {
		t.Parallel()

		clientes := &clientRepositoryFake{}
		service := NewClientService(clientes, &actorRepositoryFake{})

		_, err := service.List(context.Background(), []string{domain.RolEscribano}, "")

		if !errors.Is(err, domain.ErrNoAutorizado) {
			t.Fatalf("List() error = %v, want %v", err, domain.ErrNoAutorizado)
		}
		if clientes.listCalls != 0 {
			t.Error("List() should not reach the repository without permission")
		}
	})
}

func TestClientServiceUpdate(t *testing.T) {
	t.Parallel()

	t.Run("updates an existing client", func(t *testing.T) {
		t.Parallel()

		clientes := &clientRepositoryFake{}
		service := NewClientService(clientes, &actorRepositoryFake{usuarioID: "user-1"})

		cliente := validCliente()
		cliente.ID = "c-1"

		if _, err := service.Update(context.Background(), []string{domain.RolAdministrativo}, "sb-123", cliente); err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if clientes.updateCalls != 1 {
			t.Errorf("Update() repository calls = %d, want 1", clientes.updateCalls)
		}
		if clientes.updated.ID != "c-1" {
			t.Errorf("Update() persisted id = %q, want %q", clientes.updated.ID, "c-1")
		}
		if clientes.updatedBy != "user-1" {
			t.Errorf("Update() usuario_modificacion = %q, want %q", clientes.updatedBy, "user-1")
		}
	})

	t.Run("rejects an empty id", func(t *testing.T) {
		t.Parallel()

		clientes := &clientRepositoryFake{}
		service := NewClientService(clientes, &actorRepositoryFake{usuarioID: "user-1"})

		_, err := service.Update(context.Background(), []string{domain.RolAdministrativo}, "sb-123", validCliente())

		if !errors.Is(err, domain.ErrClienteNoEncontrado) {
			t.Fatalf("Update() error = %v, want %v", err, domain.ErrClienteNoEncontrado)
		}
		if clientes.updateCalls != 0 {
			t.Error("Update() should not reach the repository without an id")
		}
	})

	t.Run("rejects roles without permission", func(t *testing.T) {
		t.Parallel()

		clientes := &clientRepositoryFake{}
		service := NewClientService(clientes, &actorRepositoryFake{})

		cliente := validCliente()
		cliente.ID = "c-1"

		_, err := service.Update(context.Background(), []string{domain.RolAgrimensor}, "sb-123", cliente)

		if !errors.Is(err, domain.ErrNoAutorizado) {
			t.Fatalf("Update() error = %v, want %v", err, domain.ErrNoAutorizado)
		}
		if clientes.updateCalls != 0 {
			t.Error("Update() should not reach the repository without permission")
		}
	})
}

func TestClientServiceDelete(t *testing.T) {
	t.Parallel()

	t.Run("administrador gives the baja", func(t *testing.T) {
		t.Parallel()

		clientes := &clientRepositoryFake{}
		service := NewClientService(clientes, &actorRepositoryFake{usuarioID: "user-1"})

		if err := service.Delete(context.Background(), []string{domain.RolAdministrador}, "sb-123", "c-1"); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		if clientes.deleteCalls != 1 {
			t.Errorf("Delete() repository calls = %d, want 1", clientes.deleteCalls)
		}
		if clientes.deletedID != "c-1" || clientes.deletedBy != "user-1" {
			t.Errorf("Delete() called with id=%q usuario=%q", clientes.deletedID, clientes.deletedBy)
		}
	})

	t.Run("rejects every role other than administrador", func(t *testing.T) {
		t.Parallel()

		roles := []string{domain.RolAdministrativo, domain.RolInmobiliaria, domain.RolAgrimensor}

		for _, rol := range roles {
			t.Run(rol, func(t *testing.T) {
				t.Parallel()

				clientes := &clientRepositoryFake{}
				service := NewClientService(clientes, &actorRepositoryFake{usuarioID: "user-1"})

				err := service.Delete(context.Background(), []string{rol}, "sb-123", "c-1")

				if !errors.Is(err, domain.ErrNoAutorizado) {
					t.Fatalf("Delete() error = %v, want %v", err, domain.ErrNoAutorizado)
				}
				if clientes.deleteCalls != 0 {
					t.Error("Delete() should not reach the repository without permission")
				}
			})
		}
	})

	t.Run("rejects an empty id", func(t *testing.T) {
		t.Parallel()

		clientes := &clientRepositoryFake{}
		service := NewClientService(clientes, &actorRepositoryFake{usuarioID: "user-1"})

		err := service.Delete(context.Background(), []string{domain.RolAdministrador}, "sb-123", "  ")

		if !errors.Is(err, domain.ErrClienteNoEncontrado) {
			t.Fatalf("Delete() error = %v, want %v", err, domain.ErrClienteNoEncontrado)
		}
		if clientes.deleteCalls != 0 {
			t.Error("Delete() should not reach the repository without an id")
		}
	})

	t.Run("propagates repository errors", func(t *testing.T) {
		t.Parallel()

		clientes := &clientRepositoryFake{deleteErr: domain.ErrClienteNoEncontrado}
		service := NewClientService(clientes, &actorRepositoryFake{usuarioID: "user-1"})

		err := service.Delete(context.Background(), []string{domain.RolAdministrador}, "sb-123", "c-1")

		if !errors.Is(err, domain.ErrClienteNoEncontrado) {
			t.Fatalf("Delete() error = %v, want %v", err, domain.ErrClienteNoEncontrado)
		}
	})
}
