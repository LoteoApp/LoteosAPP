package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

// TestClientRepository is an integration test: it needs a real PostgreSQL
// instance with migrations applied (see docs/database.md for the Supabase
// pooler connection) and is skipped when DATABASE_URL is not set.
func TestClientRepository(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	repository := postgres.NewClientRepository(pool)
	actorID := seedUsuario(t, pool)

	t.Run("create and list", func(t *testing.T) {
		dni := newDNI(t)

		created, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Pérez", DNI: dni, Celular: "1122334455", Email: "ana@example.com",
		}, actorID)
		t.Cleanup(func() { deleteCliente(t, pool, dni) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if created.ID == "" {
			t.Error("Create() should assign an id")
		}
		if created.FechaCreacion.IsZero() {
			t.Error("Create() should set fecha_creacion")
		}
		if created.Celular != "1122334455" || created.Email != "ana@example.com" {
			t.Errorf("Create() = %#v", created)
		}

		found, err := repository.List(context.Background(), dni)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(found) != 1 || found[0].ID != created.ID {
			t.Fatalf("List() = %#v, want the created client", found)
		}
	})

	t.Run("create leaves optional contact fields empty", func(t *testing.T) {
		dni := newDNI(t)

		created, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Luis", Apellido: "Gómez", DNI: dni,
		}, actorID)
		t.Cleanup(func() { deleteCliente(t, pool, dni) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if created.Celular != "" || created.Email != "" {
			t.Errorf("Create() = %#v, want empty contact fields", created)
		}
	})

	t.Run("create rejects a duplicate dni", func(t *testing.T) {
		dni := newDNI(t)

		_, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Pérez", DNI: dni,
		}, actorID)
		t.Cleanup(func() { deleteCliente(t, pool, dni) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		_, err = repository.Create(context.Background(), domain.Cliente{
			Nombre: "Luis", Apellido: "Gómez", DNI: dni,
		}, actorID)
		if !errors.Is(err, domain.ErrDNIEnUso) {
			t.Fatalf("Create() error = %v, want %v", err, domain.ErrDNIEnUso)
		}
	})

	t.Run("list filters by name and surname", func(t *testing.T) {
		dni := newDNI(t)
		apellido := "Fernández" + dni

		created, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "María", Apellido: apellido, DNI: dni,
		}, actorID)
		t.Cleanup(func() { deleteCliente(t, pool, dni) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		found, err := repository.List(context.Background(), strings.ToLower(apellido))
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(found) != 1 || found[0].ID != created.ID {
			t.Errorf("List() = %#v, want the created client", found)
		}
	})

	t.Run("update", func(t *testing.T) {
		dni := newDNI(t)

		created, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Pérez", DNI: dni, Celular: "1122334455",
		}, actorID)
		t.Cleanup(func() { deleteCliente(t, pool, dni) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		updated, err := repository.Update(context.Background(), domain.Cliente{
			ID: created.ID, Nombre: "Ana María", Apellido: "Pérez", DNI: dni,
		}, actorID)
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.Nombre != "Ana María" {
			t.Errorf("Update() nombre = %q, want %q", updated.Nombre, "Ana María")
		}
		if updated.Celular != "" {
			t.Errorf("Update() celular = %q, want it cleared", updated.Celular)
		}
	})

	t.Run("update not found", func(t *testing.T) {
		_, err := repository.Update(context.Background(), domain.Cliente{
			ID: newUUID(t), Nombre: "Ana", Apellido: "Pérez", DNI: newDNI(t),
		}, actorID)
		if !errors.Is(err, domain.ErrClienteNoEncontrado) {
			t.Fatalf("Update() error = %v, want %v", err, domain.ErrClienteNoEncontrado)
		}
	})

	t.Run("soft delete hides the client and frees its dni", func(t *testing.T) {
		dni := newDNI(t)

		created, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Ana", Apellido: "Pérez", DNI: dni,
		}, actorID)
		t.Cleanup(func() { deleteCliente(t, pool, dni) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if err := repository.SoftDelete(context.Background(), created.ID, actorID); err != nil {
			t.Fatalf("SoftDelete() error = %v", err)
		}

		found, err := repository.List(context.Background(), dni)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(found) != 0 {
			t.Errorf("List() = %#v, want no active clients", found)
		}

		// clientes_dni_idx only covers active rows, so the DNI is reusable.
		if _, err := repository.Create(context.Background(), domain.Cliente{
			Nombre: "Luis", Apellido: "Gómez", DNI: dni,
		}, actorID); err != nil {
			t.Fatalf("Create() after baja error = %v", err)
		}
	})

	t.Run("soft delete not found", func(t *testing.T) {
		err := repository.SoftDelete(context.Background(), newUUID(t), actorID)
		if !errors.Is(err, domain.ErrClienteNoEncontrado) {
			t.Fatalf("SoftDelete() error = %v, want %v", err, domain.ErrClienteNoEncontrado)
		}
	})
}

// seedUsuario creates the user referenced by clientes.usuario_modificacion.
func seedUsuario(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	authProviderID := newUUID(t)
	usuario, err := postgres.NewUserRepository(pool).Create(context.Background(), domain.Usuario{
		AuthProviderID: authProviderID,
		Email:          newEmail(t),
		Rol:            domain.RolAdministrador,
	})
	if err != nil {
		t.Fatalf("seed usuario: %v", err)
	}
	t.Cleanup(func() { deleteUsuario(t, pool, authProviderID) })

	return usuario.ID
}

func deleteCliente(t *testing.T, pool *pgxpool.Pool, dni string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM clientes WHERE dni = $1`, dni); err != nil {
		t.Errorf("cleanup delete cliente: %v", err)
	}
}

func newDNI(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test-%s", newUUID(t))
}
