package postgres_test

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

// TestUserRepository is an integration test: it needs a real PostgreSQL
// instance with migrations applied (see docs/database.md for the Supabase
// pooler connection) and is skipped when DATABASE_URL is not set.
func TestUserRepository(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping postgres integration test")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("pgxpool.New() error = %v", err)
	}
	t.Cleanup(pool.Close)

	repository := postgres.NewUserRepository(pool)

	t.Run("create and find by auth provider id", func(t *testing.T) {
		authProviderID := newUUID(t)
		email := newEmail(t)

		created, err := repository.Create(context.Background(), domain.Usuario{
			AuthProviderID: authProviderID,
			Email:          email,
			Rol:            domain.RolAdministrativo,
		})
		t.Cleanup(func() { deleteUsuario(t, pool, authProviderID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if created.ID == "" {
			t.Error("Create() should assign an id")
		}
		if created.PerfilCompleto {
			t.Error("Create() should not mark the profile as complete")
		}
		if created.CreatedAt.IsZero() {
			t.Error("Create() should set created_at")
		}

		found, err := repository.FindByAuthProviderID(context.Background(), authProviderID)
		if err != nil {
			t.Fatalf("FindByAuthProviderID() error = %v", err)
		}
		if found.Email != email || found.Rol != domain.RolAdministrativo {
			t.Errorf("FindByAuthProviderID() = %#v", found)
		}
	})

	t.Run("create rejects duplicate email", func(t *testing.T) {
		email := newEmail(t)
		firstAuthProviderID := newUUID(t)

		_, err := repository.Create(context.Background(), domain.Usuario{
			AuthProviderID: firstAuthProviderID,
			Email:          email,
			Rol:            domain.RolAdministrativo,
		})
		t.Cleanup(func() { deleteUsuario(t, pool, firstAuthProviderID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		_, err = repository.Create(context.Background(), domain.Usuario{
			AuthProviderID: newUUID(t),
			Email:          email,
			Rol:            domain.RolAgrimensor,
		})
		if !errors.Is(err, domain.ErrEmailEnUso) {
			t.Fatalf("Create() error = %v, want %v", err, domain.ErrEmailEnUso)
		}
	})

	t.Run("find by auth provider id not found", func(t *testing.T) {
		_, err := repository.FindByAuthProviderID(context.Background(), newUUID(t))
		if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			t.Fatalf("FindByAuthProviderID() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
		}
	})

	t.Run("update profile", func(t *testing.T) {
		authProviderID := newUUID(t)

		_, err := repository.Create(context.Background(), domain.Usuario{
			AuthProviderID: authProviderID,
			Email:          newEmail(t),
			Rol:            domain.RolEscribano,
		})
		t.Cleanup(func() { deleteUsuario(t, pool, authProviderID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		updated, err := repository.UpdateProfile(context.Background(), authProviderID, "Ana", "Gómez")
		if err != nil {
			t.Fatalf("UpdateProfile() error = %v", err)
		}
		if !updated.PerfilCompleto {
			t.Error("UpdateProfile() should mark the profile as complete")
		}
		if updated.Nombre != "Ana" || updated.Apellido != "Gómez" {
			t.Errorf("UpdateProfile() = %#v", updated)
		}
	})

	t.Run("update profile not found", func(t *testing.T) {
		_, err := repository.UpdateProfile(context.Background(), newUUID(t), "Ana", "Gómez")
		if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			t.Fatalf("UpdateProfile() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
		}
	})
}

func deleteUsuario(t *testing.T, pool *pgxpool.Pool, authProviderID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `DELETE FROM usuarios WHERE auth_provider_id = $1::uuid`, authProviderID); err != nil {
		t.Errorf("cleanup delete usuario: %v", err)
	}
}

func newUUID(t *testing.T) string {
	t.Helper()

	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("generate uuid: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func newEmail(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("%s@example.com", newUUID(t))
}
