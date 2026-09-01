package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// The entity model is migration 00005. Rolling back to the version before it
// has to be explicit: a plain Down() only reverts the newest migration, so the
// assertion below would silently stop testing anything as soon as a later
// migration is added.
const entityModelMigrationVersion = 5

func TestEntityModelStateHistory(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping migration integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { db.Close() })

	schemaName := newMigrationTestSchema(t)
	schema := pgx.Identifier{schemaName}.Sanitize()
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
	})
	if _, err := db.ExecContext(ctx, "SET search_path TO "+schema); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	migrationsDir, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "migrations"))
	if err != nil {
		t.Fatalf("resolve migrations directory: %v", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(migrationsDir))
	if err != nil {
		t.Fatalf("goose.NewProvider() error = %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	seedEntityModelStateFixtures(t, ctx, db)

	t.Run("seeds initial state", func(t *testing.T) {
		var state string
		var historyRows int
		err := db.QueryRowContext(ctx, `
			SELECT r.estado_actual, count(re.id)
			FROM reservas r
			JOIN reserva_estados re ON re.reserva_id = r.id
			WHERE r.id = '00000000-0000-0000-0000-000000000030'
			GROUP BY r.estado_actual
		`).Scan(&state, &historyRows)
		if err != nil {
			t.Fatalf("query initial reservation state: %v", err)
		}
		if state != "activa" || historyRows != 1 {
			t.Fatalf("initial state = %q with %d history rows, want activa with 1", state, historyRows)
		}
	})

	t.Run("rejects direct current state update", func(t *testing.T) {
		_, err := db.ExecContext(ctx, `
			UPDATE reservas
			SET estado_actual = 'cancelada'
			WHERE id = '00000000-0000-0000-0000-000000000030'
		`)
		assertIntegrityViolation(t, err)

		if _, err := db.ExecContext(ctx, "SET loteosapp.apply_estado_actual = 'true'"); err != nil {
			t.Fatalf("set legacy bypass variable: %v", err)
		}
		_, err = db.ExecContext(ctx, `
			UPDATE reservas
			SET estado_actual = 'cancelada'
			WHERE id = '00000000-0000-0000-0000-000000000030'
		`)
		assertIntegrityViolation(t, err)
	})

	t.Run("applies inserted history state", func(t *testing.T) {
		_, err := db.ExecContext(ctx, `
			INSERT INTO reserva_estados (id, reserva_id, estado)
			VALUES (
				'00000000-0000-0000-0000-000000000031',
				'00000000-0000-0000-0000-000000000030',
				'vencida'
			)
		`)
		if err != nil {
			t.Fatalf("insert reservation history: %v", err)
		}

		var state string
		if err := db.QueryRowContext(ctx, `
			SELECT estado_actual
			FROM reservas
			WHERE id = '00000000-0000-0000-0000-000000000030'
		`).Scan(&state); err != nil {
			t.Fatalf("query current reservation state: %v", err)
		}
		if state != "vencida" {
			t.Fatalf("estado_actual = %q, want vencida", state)
		}
	})

	t.Run("keeps history append only", func(t *testing.T) {
		_, err := db.ExecContext(ctx, `
			UPDATE reserva_estados
			SET estado = 'cancelada'
			WHERE id = '00000000-0000-0000-0000-000000000031'
		`)
		assertIntegrityViolation(t, err)

		_, err = db.ExecContext(ctx, `
			DELETE FROM reserva_estados
			WHERE id = '00000000-0000-0000-0000-000000000031'
		`)
		assertIntegrityViolation(t, err)

		_, err = db.ExecContext(ctx, "TRUNCATE reserva_estados")
		assertIntegrityViolation(t, err)
	})

	t.Run("protects sale state history", func(t *testing.T) {
		var state string
		var historyRows int
		err := db.QueryRowContext(ctx, `
			SELECT v.estado_actual, count(ve.id)
			FROM ventas v
			JOIN venta_estados ve ON ve.venta_id = v.id
			WHERE v.id = '00000000-0000-0000-0000-000000000040'
			GROUP BY v.estado_actual
		`).Scan(&state, &historyRows)
		if err != nil {
			t.Fatalf("query initial sale state: %v", err)
		}
		if state != "activa" || historyRows != 1 {
			t.Fatalf("initial sale state = %q with %d history rows, want activa with 1", state, historyRows)
		}

		_, err = db.ExecContext(ctx, `
			UPDATE ventas
			SET estado_actual = 'cancelada'
			WHERE id = '00000000-0000-0000-0000-000000000040'
		`)
		assertIntegrityViolation(t, err)

		_, err = db.ExecContext(ctx, `
			INSERT INTO venta_estados (id, venta_id, estado)
			VALUES (
				'00000000-0000-0000-0000-000000000041',
				'00000000-0000-0000-0000-000000000040',
				'completada'
			)
		`)
		if err != nil {
			t.Fatalf("insert sale history: %v", err)
		}

		if err := db.QueryRowContext(ctx, `
			SELECT estado_actual
			FROM ventas
			WHERE id = '00000000-0000-0000-0000-000000000040'
		`).Scan(&state); err != nil {
			t.Fatalf("query current sale state: %v", err)
		}
		if state != "completada" {
			t.Fatalf("sale estado_actual = %q, want completada", state)
		}

		_, err = db.ExecContext(ctx, `
			UPDATE venta_estados
			SET estado = 'cancelada'
			WHERE id = '00000000-0000-0000-0000-000000000041'
		`)
		assertIntegrityViolation(t, err)

		_, err = db.ExecContext(ctx, `
			DELETE FROM venta_estados
			WHERE id = '00000000-0000-0000-0000-000000000041'
		`)
		assertIntegrityViolation(t, err)

		_, err = db.ExecContext(ctx, "TRUNCATE venta_estados")
		assertIntegrityViolation(t, err)
	})

	if _, err := provider.DownTo(ctx, entityModelMigrationVersion-1); err != nil {
		t.Fatalf("roll back entity model migration: %v", err)
	}
	var entityModelRemoved bool
	if err := db.QueryRowContext(ctx, "SELECT to_regclass($1) IS NULL", schemaName+".reservas").Scan(&entityModelRemoved); err != nil {
		t.Fatalf("check entity model rollback: %v", err)
	}
	if !entityModelRemoved {
		t.Fatal("reservas still exists after rolling back entity model migration")
	}
}

func seedEntityModelStateFixtures(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	_, err := db.ExecContext(ctx, `
		INSERT INTO usuarios (id, auth_provider_id, email, rol) VALUES
			('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'admin@example.test', 'administrador');
		INSERT INTO loteos (id, nombre) VALUES
			('00000000-0000-0000-0000-000000000010', 'A');
		INSERT INTO manzanas (id, loteo_id) VALUES
			('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000010');
		INSERT INTO lotes (id, manzana_id, loteo_id) VALUES
			('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000010'),
			('00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000010');
		INSERT INTO clientes (id, nombre, apellido, dni) VALUES
			('00000000-0000-0000-0000-000000000020', 'Test', 'Client', '1');
		INSERT INTO reservas (id, lote_id, cliente_id, vendedor_id, usuario_alta, fecha_vencimiento) VALUES
			(
				'00000000-0000-0000-0000-000000000030',
				'00000000-0000-0000-0000-000000000012',
				'00000000-0000-0000-0000-000000000020',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000001',
				now() + interval '15 days'
			);
		INSERT INTO ventas (id, lote_id, cliente_id, modalidad_pago, monto, moneda, vendedor_id, usuario_alta) VALUES
			(
				'00000000-0000-0000-0000-000000000040',
				'00000000-0000-0000-0000-000000000013',
				'00000000-0000-0000-0000-000000000020',
				'contado',
				100,
				'ARS',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000001'
			);
	`)
	if err != nil {
		t.Fatalf("seed entity model fixtures: %v", err)
	}
}

func assertIntegrityViolation(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("operation succeeded, want integrity constraint violation")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23000" {
		t.Fatalf("operation error = %v, want SQLSTATE 23000", err)
	}
}

func newMigrationTestSchema(t *testing.T) string {
	t.Helper()

	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatalf("generate test schema suffix: %v", err)
	}
	return "migration_test_" + hex.EncodeToString(suffix[:])
}
