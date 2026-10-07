package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"
)

func TestSaleReservationLinkMigration(t *testing.T) {
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
	if _, err := provider.UpTo(ctx, 14); err != nil {
		t.Fatalf("apply migrations through version 14: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO usuarios (id, auth_provider_id, email, rol) VALUES
			('00000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'admin@example.test', 'administrador');
		INSERT INTO loteos (id, nombre) VALUES ('00000000-0000-0000-0000-000000000010', 'A');
		INSERT INTO manzanas (id, loteo_id) VALUES ('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000010');
		INSERT INTO lotes (id, manzana_id, loteo_id, numero) VALUES
			('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000010', '1'),
			('00000000-0000-0000-0000-000000000013', '00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000010', '2'),
			('00000000-0000-0000-0000-000000000014', '00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000010', '3');
		INSERT INTO clientes (id, nombre, apellido, dni) VALUES ('00000000-0000-0000-0000-000000000020', 'Test', 'Client', '1');
		INSERT INTO reservas (id, lote_id, cliente_id, vendedor_id, usuario_alta, fecha_vencimiento) VALUES (
			'00000000-0000-0000-0000-000000000030', '00000000-0000-0000-0000-000000000012',
			'00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000001',
			'00000000-0000-0000-0000-000000000001', now() + interval '15 days'
		);
		INSERT INTO ventas (id, lote_id, cliente_id, modalidad_pago, monto, moneda, vendedor_id, usuario_alta) VALUES (
			'00000000-0000-0000-0000-000000000040', '00000000-0000-0000-0000-000000000013',
			'00000000-0000-0000-0000-000000000020', 'contado', 100, 'ARS',
			'00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001'
		);
	`); err != nil {
		t.Fatalf("seed rows before the link migration: %v", err)
	}

	if _, err := provider.UpTo(ctx, 15); err != nil {
		t.Fatalf("apply the link migration: %v", err)
	}

	var existingLink sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT reserva_id::text FROM ventas WHERE id = '00000000-0000-0000-0000-000000000040'`).Scan(&existingLink); err != nil {
		t.Fatalf("read existing venta link: %v", err)
	}
	if existingLink.Valid {
		t.Fatalf("existing venta link = %q, want NULL", existingLink.String)
	}

	insertLinked := func(id, lotID string) error {
		_, err := db.ExecContext(ctx, `
			INSERT INTO ventas (id, lote_id, cliente_id, modalidad_pago, monto, moneda, vendedor_id, usuario_alta, reserva_id)
			VALUES ($1::uuid, $2::uuid, '00000000-0000-0000-0000-000000000020', 'contado', 100, 'ARS',
				'00000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001',
				'00000000-0000-0000-0000-000000000030')
		`, id, lotID)
		return err
	}
	assertForeignKeyViolation(t, insertLinked("00000000-0000-0000-0000-000000000041", "00000000-0000-0000-0000-000000000014"))
	if err := insertLinked("00000000-0000-0000-0000-000000000042", "00000000-0000-0000-0000-000000000012"); err != nil {
		t.Fatalf("insert venta linked to its reserva: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO venta_estados (venta_id, estado) VALUES ('00000000-0000-0000-0000-000000000042', 'cancelada')
	`); err != nil {
		t.Fatalf("cancel linked venta: %v", err)
	}
	assertUniqueViolation(t, insertLinked("00000000-0000-0000-0000-000000000043", "00000000-0000-0000-0000-000000000012"))

	_, err = db.ExecContext(ctx, `UPDATE ventas SET reserva_id = NULL WHERE id = '00000000-0000-0000-0000-000000000042'`)
	assertIntegrityViolation(t, err)
	if _, err := db.ExecContext(ctx, `UPDATE ventas SET fecha_modificacion = now() WHERE id = '00000000-0000-0000-0000-000000000042'`); err != nil {
		t.Fatalf("update another venta field: %v", err)
	}

	if _, err := provider.DownTo(ctx, 14); err != nil {
		t.Fatalf("roll back the link migration: %v", err)
	}
	var columnRemoved bool
	if err := db.QueryRowContext(ctx, `
		SELECT NOT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = 'ventas' AND column_name = 'reserva_id'
		)
	`).Scan(&columnRemoved); err != nil {
		t.Fatalf("check link rollback: %v", err)
	}
	if !columnRemoved {
		t.Fatal("ventas.reserva_id still exists after rolling back")
	}
	var ventas int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM ventas`).Scan(&ventas); err != nil {
		t.Fatalf("count ventas after rollback: %v", err)
	}
	if ventas != 2 {
		t.Fatalf("ventas after rollback = %d, want the 2 rows kept", ventas)
	}
}
