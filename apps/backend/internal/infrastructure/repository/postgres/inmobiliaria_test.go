package postgres_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

func inmobiliariaStrPtr(s string) *string {
	return &s
}

// newCUIT builds an 11-digit value that is unique per test run, so the
// unique index on active inmobiliarias only fires when a test means it to.
func newCUIT(t *testing.T) string {
	t.Helper()

	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, newUUID(t))
	if len(digits) < 11 {
		t.Fatalf("generated uuid has %d digits, want at least 11", len(digits))
	}

	return digits[:11]
}

// TestInmobiliariaRepository is an integration test: it needs a real
// PostgreSQL instance with migrations applied (see docs/database.md) and is
// skipped when DATABASE_URL is not set. CI sets it, so this runs on every
// pull request; locally, run it with
// `DATABASE_URL=... go test ./internal/infrastructure/repository/postgres/...`.
func TestInmobiliariaRepository(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	repository := postgres.NewInmobiliariaRepository(pool)

	t.Run("create and list", func(t *testing.T) {
		razonSocial := "Lotes del Sur " + newUUID(t)
		cuit := newCUIT(t)
		telefono := "3415551234"
		email := "contacto@lotesdelsur.com"

		created, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: razonSocial, CUIT: &cuit, Telefono: &telefono, Email: &email,
			UsuarioModificacion: seedUsuario(t, pool),
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if created.ID == "" {
			t.Error("Create() should assign an id")
		}
		if created.FechaCreacion.IsZero() {
			t.Error("Create() should set fecha_creacion")
		}

		found, err := repository.List(context.Background(), razonSocial)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(found) != 1 || found[0].ID != created.ID {
			t.Errorf("List() = %#v", found)
		}
	})

	t.Run("list finds an agency by cuit", func(t *testing.T) {
		cuit := newCUIT(t)
		created, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Lotes del Sur " + newUUID(t), CUIT: &cuit,
			UsuarioModificacion: seedUsuario(t, pool),
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		found, err := repository.List(context.Background(), cuit)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(found) != 1 || found[0].ID != created.ID {
			t.Errorf("List(%q) = %#v", cuit, found)
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
		created, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial:         "Lotes del Sur " + newUUID(t),
			UsuarioModificacion: seedUsuario(t, pool),
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		for _, search := range []string{"%", "_"} {
			found, err := repository.List(context.Background(), search)
			if err != nil {
				t.Fatalf("List(%q) error = %v", search, err)
			}
			for _, inmobiliaria := range found {
				if inmobiliaria.ID == created.ID {
					t.Errorf("List(%q) matched every inmobiliaria, want the wildcard treated as a literal", search)
				}
			}
		}
	})

	t.Run("create rejects duplicate cuit among active agencies", func(t *testing.T) {
		cuit := newCUIT(t)
		actor := seedUsuario(t, pool)

		first, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Lotes del Sur " + newUUID(t), CUIT: &cuit, UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, first.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		_, err = repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Altamira " + newUUID(t), CUIT: &cuit, UsuarioModificacion: actor,
		})
		if !errors.Is(err, domain.ErrCUITEnUso) {
			t.Fatalf("Create() error = %v, want %v", err, domain.ErrCUITEnUso)
		}
	})

	// The unique index is partial on cuit IS NOT NULL, so agencies without
	// one must not collide with each other.
	t.Run("create allows several agencies without cuit", func(t *testing.T) {
		actor := seedUsuario(t, pool)

		first, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Lotes del Sur " + newUUID(t), UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, first.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		second, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Altamira " + newUUID(t), UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, second.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v, want a second agency without cuit to be allowed", err)
		}
	})

	t.Run("cuit can be reused after baja", func(t *testing.T) {
		cuit := newCUIT(t)
		actor := seedUsuario(t, pool)

		first, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Lotes del Sur " + newUUID(t), CUIT: &cuit, UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, first.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if err := repository.SoftDelete(context.Background(), first.ID, actor); err != nil {
			t.Fatalf("SoftDelete() error = %v", err)
		}

		second, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Altamira " + newUUID(t), CUIT: &cuit, UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, second.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v after baja, want nil", err)
		}
	})

	t.Run("update replaces the fields that are present", func(t *testing.T) {
		actor := seedUsuario(t, pool)
		created, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Lotes del Sur " + newUUID(t), UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		newCUITValue := newCUIT(t)
		updated, err := repository.Update(context.Background(), domain.InmobiliariaUpdate{
			ID: created.ID, RazonSocial: inmobiliariaStrPtr("Lotes del Sur SRL"),
			CUIT: &newCUITValue, UsuarioModificacion: actor,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.RazonSocial != "Lotes del Sur SRL" {
			t.Errorf("Update() razon social = %q, want %q", updated.RazonSocial, "Lotes del Sur SRL")
		}
		if updated.CUIT == nil || *updated.CUIT != newCUITValue {
			t.Errorf("Update() cuit = %v, want %q", updated.CUIT, newCUITValue)
		}
	})

	t.Run("update leaves omitted fields unchanged", func(t *testing.T) {
		actor := seedUsuario(t, pool)
		cuit := newCUIT(t)
		telefono := "3415551234"
		created, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Lotes del Sur " + newUUID(t), CUIT: &cuit, Telefono: &telefono,
			UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Only email is sent; razon social, cuit and telefono are omitted
		// and must survive the update untouched.
		updated, err := repository.Update(context.Background(), domain.InmobiliariaUpdate{
			ID: created.ID, Email: inmobiliariaStrPtr("contacto@lotesdelsur.com"), UsuarioModificacion: actor,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.RazonSocial != created.RazonSocial {
			t.Errorf("Update() razon social = %q, want it unchanged (%q)", updated.RazonSocial, created.RazonSocial)
		}
		if updated.CUIT == nil || *updated.CUIT != cuit {
			t.Errorf("Update() cuit = %v, want it unchanged (%q)", updated.CUIT, cuit)
		}
		if updated.Telefono == nil || *updated.Telefono != telefono {
			t.Errorf("Update() telefono = %v, want it unchanged (%q)", updated.Telefono, telefono)
		}
		if updated.Email == nil || *updated.Email != "contacto@lotesdelsur.com" {
			t.Errorf("Update() email = %v, want it replaced", updated.Email)
		}
	})

	t.Run("update rejects duplicate cuit among active agencies", func(t *testing.T) {
		actor := seedUsuario(t, pool)
		takenCUIT := newCUIT(t)

		first, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Lotes del Sur " + newUUID(t), CUIT: &takenCUIT, UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, first.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		second, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Altamira " + newUUID(t), UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, second.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		_, err = repository.Update(context.Background(), domain.InmobiliariaUpdate{
			ID: second.ID, CUIT: &takenCUIT, UsuarioModificacion: actor,
		})
		if !errors.Is(err, domain.ErrCUITEnUso) {
			t.Fatalf("Update() error = %v, want %v", err, domain.ErrCUITEnUso)
		}
	})

	t.Run("update not found", func(t *testing.T) {
		_, err := repository.Update(context.Background(), domain.InmobiliariaUpdate{
			ID: newUUID(t), RazonSocial: inmobiliariaStrPtr("Lotes del Sur"), UsuarioModificacion: seedUsuario(t, pool),
		})
		if !errors.Is(err, domain.ErrInmobiliariaNoEncontrada) {
			t.Fatalf("Update() error = %v, want %v", err, domain.ErrInmobiliariaNoEncontrada)
		}
	})

	t.Run("soft delete", func(t *testing.T) {
		actor := seedUsuario(t, pool)
		razonSocial := "Lotes del Sur " + newUUID(t)
		created, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: razonSocial, UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if err := repository.SoftDelete(context.Background(), created.ID, actor); err != nil {
			t.Fatalf("SoftDelete() error = %v", err)
		}

		found, err := repository.List(context.Background(), razonSocial)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(found) != 0 {
			t.Errorf("List() should not return agencies given de baja, got %#v", found)
		}
	})

	t.Run("soft delete not found", func(t *testing.T) {
		err := repository.SoftDelete(context.Background(), newUUID(t), seedUsuario(t, pool))
		if !errors.Is(err, domain.ErrInmobiliariaNoEncontrada) {
			t.Fatalf("SoftDelete() error = %v, want %v", err, domain.ErrInmobiliariaNoEncontrada)
		}
	})

	// An agency given de baja must not be editable either: the update only
	// matches rows with fecha_baja IS NULL.
	t.Run("update of an agency given de baja is reported as not found", func(t *testing.T) {
		actor := seedUsuario(t, pool)
		created, err := repository.Create(context.Background(), domain.Inmobiliaria{
			RazonSocial: "Lotes del Sur " + newUUID(t), UsuarioModificacion: actor,
		})
		t.Cleanup(func() { deleteInmobiliaria(t, pool, created.ID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if err := repository.SoftDelete(context.Background(), created.ID, actor); err != nil {
			t.Fatalf("SoftDelete() error = %v", err)
		}

		_, err = repository.Update(context.Background(), domain.InmobiliariaUpdate{
			ID: created.ID, RazonSocial: inmobiliariaStrPtr("Lotes del Sur SRL"), UsuarioModificacion: actor,
		})
		if !errors.Is(err, domain.ErrInmobiliariaNoEncontrada) {
			t.Fatalf("Update() error = %v, want %v", err, domain.ErrInmobiliariaNoEncontrada)
		}
	})
}

func deleteInmobiliaria(t *testing.T, pool *pgxpool.Pool, id string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM inmobiliarias WHERE id = $1::uuid`, id); err != nil {
		t.Errorf("cleanup delete inmobiliaria: %v", err)
	}
}
