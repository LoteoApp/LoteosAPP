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

	t.Run("create stores nombre, apellido and perfil completo", func(t *testing.T) {
		authProviderID := newUUID(t)

		created, err := repository.Create(context.Background(), domain.Usuario{
			AuthProviderID: authProviderID,
			Email:          newEmail(t),
			Nombre:         "Ana",
			Apellido:       "Gómez",
			Rol:            domain.RolAgrimensor,
			PerfilCompleto: true,
		})
		t.Cleanup(func() { deleteUsuario(t, pool, authProviderID) })
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if created.Nombre != "Ana" || created.Apellido != "Gómez" || !created.PerfilCompleto {
			t.Errorf("Create() = %#v", created)
		}
		if !created.Activo() {
			t.Error("Create() should leave the user active")
		}
	})

	t.Run("find by id", func(t *testing.T) {
		agrimensor := createAgrimensor(t, pool, repository)

		found, err := repository.FindByID(context.Background(), agrimensor.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if found.ID != agrimensor.ID || !found.EsAgrimensor() {
			t.Errorf("FindByID() = %#v", found)
		}
	})

	t.Run("find by id not found", func(t *testing.T) {
		_, err := repository.FindByID(context.Background(), newUUID(t))
		if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			t.Fatalf("FindByID() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
		}
	})

	t.Run("list by rol excludes other roles and, by default, the ones given de baja", func(t *testing.T) {
		agrimensor := createAgrimensor(t, pool, repository)
		administrativo := createUsuarioConRol(t, pool, repository, domain.RolAdministrativo, "Zoe", "Vera")

		dadoDeBaja := createAgrimensor(t, pool, repository)
		if err := repository.SoftDelete(context.Background(), dadoDeBaja.ID, agrimensor.ID); err != nil {
			t.Fatalf("SoftDelete() error = %v", err)
		}

		activos, err := repository.ListByRol(context.Background(), domain.RolAgrimensor, false)
		if err != nil {
			t.Fatalf("ListByRol() error = %v", err)
		}
		if !containsUsuario(activos, agrimensor.ID) {
			t.Error("ListByRol() should return the active agrimensor")
		}
		if containsUsuario(activos, dadoDeBaja.ID) {
			t.Error("ListByRol() should not return an agrimensor given de baja")
		}
		if containsUsuario(activos, administrativo.ID) {
			t.Error("ListByRol() should not return a user of another rol")
		}

		todos, err := repository.ListByRol(context.Background(), domain.RolAgrimensor, true)
		if err != nil {
			t.Fatalf("ListByRol() error = %v", err)
		}
		if !containsUsuario(todos, dadoDeBaja.ID) {
			t.Error("ListByRol(includeInactive) should return the agrimensor given de baja")
		}
	})

	t.Run("update applies only the fields sent", func(t *testing.T) {
		agrimensor := createAgrimensor(t, pool, repository)
		nombre := "Ana María"

		updated, err := repository.Update(context.Background(), domain.UsuarioUpdate{
			ID:                  agrimensor.ID,
			Nombre:              &nombre,
			UsuarioModificacion: agrimensor.ID,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}
		if updated.Nombre != nombre {
			t.Errorf("Update() nombre = %q, want %q", updated.Nombre, nombre)
		}
		if updated.Apellido != agrimensor.Apellido {
			t.Errorf("Update() should leave apellido as %q, got %q", agrimensor.Apellido, updated.Apellido)
		}
		if updated.Email != agrimensor.Email {
			t.Errorf("Update() should not change the email, got %q", updated.Email)
		}
	})

	t.Run("update not found", func(t *testing.T) {
		nombre := "Ana"
		actor := createAgrimensor(t, pool, repository)

		_, err := repository.Update(context.Background(), domain.UsuarioUpdate{
			ID:                  newUUID(t),
			Nombre:              &nombre,
			UsuarioModificacion: actor.ID,
		})
		if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			t.Fatalf("Update() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
		}
	})

	t.Run("soft delete keeps the row and marks the fecha de baja", func(t *testing.T) {
		agrimensor := createAgrimensor(t, pool, repository)
		actor := createAgrimensor(t, pool, repository)

		if err := repository.SoftDelete(context.Background(), agrimensor.ID, actor.ID); err != nil {
			t.Fatalf("SoftDelete() error = %v", err)
		}

		found, err := repository.FindByID(context.Background(), agrimensor.ID)
		if err != nil {
			t.Fatalf("FindByID() error = %v", err)
		}
		if found.Activo() {
			t.Error("SoftDelete() should leave the user inactive")
		}

		if err := repository.SoftDelete(context.Background(), agrimensor.ID, actor.ID); !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			t.Fatalf("SoftDelete() on an inactive user error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
		}
	})

	t.Run("update ignores a user given de baja", func(t *testing.T) {
		agrimensor := createAgrimensor(t, pool, repository)
		actor := createAgrimensor(t, pool, repository)
		if err := repository.SoftDelete(context.Background(), agrimensor.ID, actor.ID); err != nil {
			t.Fatalf("SoftDelete() error = %v", err)
		}

		nombre := "Ana María"
		_, err := repository.Update(context.Background(), domain.UsuarioUpdate{
			ID:                  agrimensor.ID,
			Nombre:              &nombre,
			UsuarioModificacion: actor.ID,
		})
		if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			t.Fatalf("Update() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
		}
	})
}

func createAgrimensor(t *testing.T, pool *pgxpool.Pool, repository *postgres.UserRepository) domain.Usuario {
	t.Helper()
	return createUsuarioConRol(t, pool, repository, domain.RolAgrimensor, "Ana", "Gómez")
}

func createUsuarioConRol(
	t *testing.T,
	pool *pgxpool.Pool,
	repository *postgres.UserRepository,
	rol domain.Rol,
	nombre, apellido string,
) domain.Usuario {
	t.Helper()

	authProviderID := newUUID(t)
	created, err := repository.Create(context.Background(), domain.Usuario{
		AuthProviderID: authProviderID,
		Email:          newEmail(t),
		Nombre:         nombre,
		Apellido:       apellido,
		Rol:            rol,
		PerfilCompleto: true,
	})
	t.Cleanup(func() { deleteUsuario(t, pool, authProviderID) })
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	return created
}

func containsUsuario(usuarios []domain.Usuario, id string) bool {
	for _, usuario := range usuarios {
		if usuario.ID == id {
			return true
		}
	}

	return false
}

func deleteUsuario(t *testing.T, pool *pgxpool.Pool, authProviderID string) {
	t.Helper()

	// usuario_modificacion points at usuarios, so the audit trail another
	// test row left behind has to be cleared before this row can go.
	if _, err := pool.Exec(context.Background(), `
		UPDATE usuarios
		SET usuario_modificacion = NULL
		WHERE usuario_modificacion = (SELECT id FROM usuarios WHERE auth_provider_id = $1::uuid)
	`, authProviderID); err != nil {
		t.Errorf("cleanup clear usuario_modificacion: %v", err)
	}
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
