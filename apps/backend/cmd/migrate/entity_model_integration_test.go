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
	if _, err := provider.UpTo(ctx, 7); err != nil {
		t.Fatalf("apply migrations through version 7: %v", err)
	}

	var activeDxfIndexPresent bool
	if err := db.QueryRowContext(
		ctx,
		"SELECT to_regclass($1) IS NOT NULL",
		schemaName+".archivos_loteo_active_dxf_idx",
	).Scan(&activeDxfIndexPresent); err != nil {
		t.Fatalf("check active DXF index: %v", err)
	}
	if !activeDxfIndexPresent {
		t.Fatal("active DXF unique index was not created")
	}

	seedEntityModelStateFixtures(t, ctx, db)

	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("apply remaining migrations: %v", err)
	}

	t.Run("backfills existing lotes and seeds new ones", func(t *testing.T) {
		assertLotState(t, ctx, db, "00000000-0000-0000-0000-000000000012", "disponible", 1)
		assertLotState(t, ctx, db, "00000000-0000-0000-0000-000000000013", "vendido", 1)

		var legacyOrigin, legacyReason string
		if err := db.QueryRowContext(ctx, `
			SELECT origen, razon FROM lote_estados
			WHERE id = '00000000-0000-0000-0000-000000000015'
		`).Scan(&legacyOrigin, &legacyReason); err != nil {
			t.Fatalf("query migrated history: %v", err)
		}
		if legacyOrigin != "sistema" || legacyReason == "" {
			t.Fatalf("migrated history origin = %q, reason = %q", legacyOrigin, legacyReason)
		}

		if _, err := db.ExecContext(ctx, `
			INSERT INTO lotes (id, manzana_id, loteo_id) VALUES (
				'00000000-0000-0000-0000-000000000014',
				'00000000-0000-0000-0000-000000000011',
				'00000000-0000-0000-0000-000000000010'
			)
		`); err != nil {
			t.Fatalf("insert lote after state migration: %v", err)
		}
		assertLotState(t, ctx, db, "00000000-0000-0000-0000-000000000014", "disponible", 1)
	})

	t.Run("protects lote current state and validates transitions", func(t *testing.T) {
		_, err := db.ExecContext(ctx, `
			UPDATE lotes SET estado_actual = 'vendido'
			WHERE id = '00000000-0000-0000-0000-000000000012'
		`)
		assertIntegrityViolation(t, err)

		_, err = db.ExecContext(ctx, `
			INSERT INTO lote_estados (lote_id, estado, origen)
			VALUES ('00000000-0000-0000-0000-000000000012', 'reservado', 'sistema')
		`)
		assertCheckViolation(t, err)

		_, err = db.ExecContext(ctx, `
			INSERT INTO lote_estados (
				id, lote_id, estado, origen, reserva_id, usuario_modificacion, fecha_creacion
			) VALUES (
				'00000000-0000-0000-0000-000000000032',
				'00000000-0000-0000-0000-000000000012',
				'reservado', 'reserva',
				'00000000-0000-0000-0000-000000000030',
				'00000000-0000-0000-0000-000000000001',
				'2000-01-01T00:00:00Z'
			)
		`)
		if err != nil {
			t.Fatalf("insert valid lote transition: %v", err)
		}
		assertLotState(t, ctx, db, "00000000-0000-0000-0000-000000000012", "reservado", 2)
		var eventTime time.Time
		if err := db.QueryRowContext(ctx, `
			SELECT fecha_creacion FROM lote_estados
			WHERE id = '00000000-0000-0000-0000-000000000032'
		`).Scan(&eventTime); err != nil {
			t.Fatalf("query lote event time: %v", err)
		}
		if eventTime.Year() == 2000 {
			t.Error("lote history should use the database clock instead of a caller-supplied timestamp")
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO lote_estados (lote_id, estado, origen)
			VALUES ('00000000-0000-0000-0000-000000000014', 'finalizado', 'sistema')
		`)
		assertCheckViolation(t, err)
		assertLotState(t, ctx, db, "00000000-0000-0000-0000-000000000014", "disponible", 1)
	})

	t.Run("rejects commercial references belonging to another lote", func(t *testing.T) {
		_, err := db.ExecContext(ctx, `
			INSERT INTO lote_estados (lote_id, estado, origen, reserva_id)
			VALUES (
				'00000000-0000-0000-0000-000000000014',
				'reservado', 'reserva',
				'00000000-0000-0000-0000-000000000030'
			)
		`)
		assertForeignKeyViolation(t, err)

		_, err = db.ExecContext(ctx, `
			INSERT INTO lote_estados (lote_id, estado, origen, venta_id)
			VALUES (
				'00000000-0000-0000-0000-000000000014',
				'vendido', 'venta',
				'00000000-0000-0000-0000-000000000040'
			)
		`)
		assertForeignKeyViolation(t, err)
		assertLotState(t, ctx, db, "00000000-0000-0000-0000-000000000014", "disponible", 1)
	})

	t.Run("requires a reason for a manual cancellation", func(t *testing.T) {
		_, err := db.ExecContext(ctx, `
			INSERT INTO lote_estados (lote_id, estado, origen, reserva_id)
			VALUES (
				'00000000-0000-0000-0000-000000000012',
				'disponible', 'reserva',
				'00000000-0000-0000-0000-000000000030'
			)
		`)
		assertCheckViolation(t, err)

		_, err = db.ExecContext(ctx, `
			INSERT INTO lote_estados (lote_id, estado, origen, razon, reserva_id)
			VALUES (
				'00000000-0000-0000-0000-000000000012',
				'disponible', 'reserva', 'El cliente desistio',
				'00000000-0000-0000-0000-000000000030'
			)
		`)
		if err != nil {
			t.Fatalf("insert justified lote cancellation: %v", err)
		}
		assertLotState(t, ctx, db, "00000000-0000-0000-0000-000000000012", "disponible", 3)
	})

	t.Run("keeps lote history append only", func(t *testing.T) {
		_, err := db.ExecContext(ctx, `
			UPDATE lote_estados SET razon = 'changed'
			WHERE id = '00000000-0000-0000-0000-000000000032'
		`)
		assertIntegrityViolation(t, err)

		_, err = db.ExecContext(ctx, `
			DELETE FROM lote_estados
			WHERE id = '00000000-0000-0000-0000-000000000032'
		`)
		assertIntegrityViolation(t, err)

		_, err = db.ExecContext(ctx, "TRUNCATE lote_estados")
		assertIntegrityViolation(t, err)
	})

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

	t.Run("protects reservation identity and idempotency", func(t *testing.T) {
		_, err := db.ExecContext(ctx, `
			UPDATE reservas
			SET fecha_vencimiento = fecha_vencimiento + interval '1 hour'
			WHERE id = '00000000-0000-0000-0000-000000000030'
		`)
		assertIntegrityViolation(t, err)

		_, err = db.ExecContext(ctx, `
			INSERT INTO reservas (
				id, lote_id, cliente_id, vendedor_id, usuario_alta,
				fecha_vencimiento, idempotency_key, idempotency_payload_hash
			) VALUES (
				'00000000-0000-0000-0000-000000000050',
				'00000000-0000-0000-0000-000000000013',
				'00000000-0000-0000-0000-000000000020',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000001',
				now() + interval '15 days', 'reservation-key',
				repeat('0', 64)
			)
		`)
		if err != nil {
			t.Fatalf("insert idempotent reservation: %v", err)
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO reserva_estados (reserva_id, estado)
			VALUES ('00000000-0000-0000-0000-000000000050', 'cancelada')
		`)
		assertCheckViolation(t, err)

		_, err = db.ExecContext(ctx, `
			INSERT INTO reservas (
				id, lote_id, cliente_id, vendedor_id, usuario_alta,
				fecha_vencimiento, idempotency_key, idempotency_payload_hash
			) VALUES (
				'00000000-0000-0000-0000-000000000051',
				'00000000-0000-0000-0000-000000000014',
				'00000000-0000-0000-0000-000000000020',
				'00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000001',
				now() + interval '15 days', 'reservation-key',
				repeat('1', 64)
			)
		`)
		assertUniqueViolation(t, err)

		_, err = db.ExecContext(ctx, `
			INSERT INTO reserva_estados (reserva_id, estado)
			VALUES ('00000000-0000-0000-0000-000000000030', 'cancelada')
		`)
		assertCheckViolation(t, err)
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

	if _, err := provider.DownTo(ctx, 7); err != nil {
		t.Fatalf("roll back lot state migration: %v", err)
	}
	var preservedHistory int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*) FROM lote_estados
		WHERE id = '00000000-0000-0000-0000-000000000015'
	`).Scan(&preservedHistory); err != nil {
		t.Fatalf("query history after lot state rollback: %v", err)
	}
	if preservedHistory != 1 {
		t.Fatalf("preexisting lot history rows after rollback = %d, want 1", preservedHistory)
	}

	if _, err := provider.DownTo(ctx, 4); err != nil {
		t.Fatalf("roll back entity model migrations: %v", err)
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
		INSERT INTO lote_estados (id, lote_id, estado) VALUES
			(
				'00000000-0000-0000-0000-000000000015',
				'00000000-0000-0000-0000-000000000013',
				'vendido'
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

func assertCheckViolation(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("operation succeeded, want check violation")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" {
		t.Fatalf("operation error = %v, want SQLSTATE 23514", err)
	}
}

func assertForeignKeyViolation(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("operation succeeded, want foreign key violation")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		t.Fatalf("operation error = %v, want SQLSTATE 23503", err)
	}
}

func assertUniqueViolation(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("operation succeeded, want unique violation")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Fatalf("operation error = %v, want SQLSTATE 23505", err)
	}
}

func assertLotState(t *testing.T, ctx context.Context, db *sql.DB, loteID, wantState string, wantEvents int) {
	t.Helper()

	var state string
	var events int
	err := db.QueryRowContext(ctx, `
		SELECT l.estado_actual, count(le.id)
		FROM lotes l
		JOIN lote_estados le ON le.lote_id = l.id
		WHERE l.id = $1::uuid
		GROUP BY l.estado_actual
	`, loteID).Scan(&state, &events)
	if err != nil {
		t.Fatalf("query lote state: %v", err)
	}
	if state != wantState || events != wantEvents {
		t.Fatalf("lote state = %q with %d events, want %q with %d", state, events, wantState, wantEvents)
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
