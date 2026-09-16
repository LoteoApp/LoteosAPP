package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

func clienteStrPtr(s string) *string {
	return &s
}

// TestClienteRepository is an integration test: it needs a real PostgreSQL
// instance with migrations applied (see docs/database.md) and is skipped
// when DATABASE_URL is not set. CI sets it, so this runs on every pull
// request; locally, run it with
// `DATABASE_URL=... go test ./internal/infrastructure/repository/postgres/...`.
func TestClienteRepository(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	repository := postgres.NewClienteRepository(pool)

	t.Run("create and list", func(t *testing.T) {
		dni := newUUID(t)
		celular := "1122334455"
		email := "ana@example.com"

		created, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Perez", DNI: dni, Celular: &celular, Email: &email,
			UsuarioModificacion: seedUsuario(t, pool),
		})
		t.Cleanup(func() { deleteCliente(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if created.ID == "" {
			t.Error("Create() should assign an id")
		}
		if created.FechaCreacion.IsZero() {
			t.Error("Create() should set fecha_creacion")
		}

		found, err := repository.List(context.Background(), dni)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(found) != 1 || found[0].ID != created.ID {
			t.Errorf("List() = %#v", found)
		}
	})

	t.Run("list with no matches returns an empty slice, not null", func(t *testing.T) {
		found, err := repository.List(context.Background(), newUUID(t))
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if found == nil {
			t.Error("List() should return an empty slice, not nil, when there are no matches")
		}
		if len(found) != 0 {
			t.Errorf("List() = %#v, want no results", found)
		}
	})

	t.Run("search treats ILIKE wildcards as literal characters", func(t *testing.T) {
		created, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Perez", DNI: newUUID(t),
			UsuarioModificacion: seedUsuario(t, pool),
		})
		t.Cleanup(func() { deleteCliente(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		for _, search := range []string{"%", "_"} {
			found, err := repository.List(context.Background(), search)
			if err != nil {
				t.Fatalf("List(%q) error = %v", search, err)
			}
			for _, cliente := range found {
				if cliente.ID == created.ID {
					t.Errorf("List(%q) matched every cliente, want the wildcard treated as a literal", search)
				}
			}
		}
	})

	t.Run("create rejects duplicate dni among active clients", func(t *testing.T) {
		dni := newUUID(t)
		actor := seedUsuario(t, pool)

		first, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Perez", DNI: dni, UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteCliente(t, pool, first.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		_, err = repository.Create(context.Background(), domain.Cliente{
			Nombre: "Beto", Apellido: "Gomez", DNI: dni, UsuarioModificacion: actor,
		})
		if !errors.Is(err, domain.ErrDNIEnUso) {
			t.Fatalf("Create() error = %v, want %v", err, domain.ErrDNIEnUso)
		}
	})

	t.Run("dni can be reused after baja", func(t *testing.T) {
		dni := newUUID(t)
		actor := seedUsuario(t, pool)

		first, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Perez", DNI: dni, UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteCliente(t, pool, first.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if err := repository.SoftDelete(context.Background(), first.ID, actor); err != nil {
			t.Fatalf("SoftDelete() error = %v", err)
		}

		second, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Beto", Apellido: "Gomez", DNI: dni, UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteCliente(t, pool, second.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v after baja, want nil", err)
		}
	})

	t.Run("update replaces the fields that are present", func(t *testing.T) {
		actor := seedUsuario(t, pool)
		created, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Perez", DNI: newUUID(t), UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteCliente(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		newDNI := newUUID(t)
		updated, err := repository.Update(context.Background(), domain.ClienteUpdate{
			ID: created.ID, Nombre: clienteStrPtr("Ana Maria"), Apellido: clienteStrPtr("Perez"), DNI: clienteStrPtr(newDNI),
			UsuarioModificacion: actor,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.Nombre != "Ana Maria" || updated.DNI != newDNI {
			t.Errorf("Update() = %#v", updated)
		}
	})

	t.Run("update leaves omitted fields unchanged", func(t *testing.T) {
		actor := seedUsuario(t, pool)
		celular := "1122334455"
		dni := newUUID(t)
		created, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Perez", DNI: dni, Celular: &celular, UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteCliente(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Only nombre is sent; apellido, dni and celular are omitted and
		// must survive the update untouched.
		updated, err := repository.Update(context.Background(), domain.ClienteUpdate{
			ID: created.ID, Nombre: clienteStrPtr("Ana Maria"), UsuarioModificacion: actor,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.Nombre != "Ana Maria" {
			t.Errorf("Update() nombre = %q, want %q", updated.Nombre, "Ana Maria")
		}
		if updated.Apellido != "Perez" {
			t.Errorf("Update() apellido = %q, want it unchanged (%q)", updated.Apellido, "Perez")
		}
		if updated.DNI != dni {
			t.Errorf("Update() dni = %q, want it unchanged (%q)", updated.DNI, dni)
		}
		if updated.Celular == nil || *updated.Celular != celular {
			t.Errorf("Update() celular = %v, want it unchanged (%q)", updated.Celular, celular)
		}
	})

	t.Run("update rejects duplicate dni among active clients", func(t *testing.T) {
		actor := seedUsuario(t, pool)
		takenDNI := newUUID(t)

		first, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Perez", DNI: takenDNI, UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteCliente(t, pool, first.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		second, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Beto", Apellido: "Gomez", DNI: newUUID(t), UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteCliente(t, pool, second.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		_, err = repository.Update(context.Background(), domain.ClienteUpdate{
			ID: second.ID, DNI: clienteStrPtr(takenDNI), UsuarioModificacion: actor,
		})
		if !errors.Is(err, domain.ErrDNIEnUso) {
			t.Fatalf("Update() error = %v, want %v", err, domain.ErrDNIEnUso)
		}
	})

	t.Run("update not found", func(t *testing.T) {
		_, err := repository.Update(context.Background(), domain.ClienteUpdate{
			ID: newUUID(t), Nombre: clienteStrPtr("Ana"), UsuarioModificacion: seedUsuario(t, pool),
		})
		if !errors.Is(err, domain.ErrClienteNoEncontrado) {
			t.Fatalf("Update() error = %v, want %v", err, domain.ErrClienteNoEncontrado)
		}
	})

	t.Run("soft delete", func(t *testing.T) {
		actor := seedUsuario(t, pool)
		created, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Perez", DNI: newUUID(t), UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteCliente(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if err := repository.SoftDelete(context.Background(), created.ID, actor); err != nil {
			t.Fatalf("SoftDelete() error = %v", err)
		}

		found, err := repository.List(context.Background(), created.DNI)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(found) != 0 {
			t.Errorf("List() should not return clients given de baja, got %#v", found)
		}
	})

	t.Run("soft delete not found", func(t *testing.T) {
		err := repository.SoftDelete(context.Background(), newUUID(t), seedUsuario(t, pool))
		if !errors.Is(err, domain.ErrClienteNoEncontrado) {
			t.Fatalf("SoftDelete() error = %v, want %v", err, domain.ErrClienteNoEncontrado)
		}
	})
}

func seedUsuario(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	authProviderID := newUUID(t)
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO usuarios (auth_provider_id, email, rol)
		VALUES ($1::uuid, $2, $3)
		RETURNING id::text
	`, authProviderID, newEmail(t), domain.RolAdministrador).Scan(&id)
	if err != nil {
		t.Fatalf("seed usuario: %v", err)
	}
	t.Cleanup(func() { deleteUsuario(t, pool, authProviderID) })

	return id
}

func deleteCliente(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM clientes WHERE id = $1::uuid`, id); err != nil {
		t.Errorf("cleanup delete cliente: %v", err)
	}
}
