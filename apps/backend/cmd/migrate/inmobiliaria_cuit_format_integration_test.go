package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"
)

func newCuitMigrationProvider(t *testing.T, ctx context.Context) (*sql.DB, *goose.Provider) {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set, skipping migration integration test")
	}

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
	if _, err := provider.UpTo(ctx, 15); err != nil {
		t.Fatalf("apply migrations through version 15: %v", err)
	}

	return db, provider
}

func TestInmobiliariaCuitFormatMigration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, provider := newCuitMigrationProvider(t, ctx)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO inmobiliarias (id, razon_social, cuit, fecha_baja) VALUES
			('00000000-0000-0000-0000-000000000001', 'Con guiones', '30-71234567-8', NULL),
			('00000000-0000-0000-0000-000000000002', 'Con puntos de baja', '20.12345678.9', now()),
			('00000000-0000-0000-0000-000000000003', 'Sin cuit', NULL, NULL),
			('00000000-0000-0000-0000-000000000004', 'Ya normalizada', '30709876543', NULL);
	`); err != nil {
		t.Fatalf("seed inmobiliarias before the migration: %v", err)
	}

	if _, err := provider.UpTo(ctx, 16); err != nil {
		t.Fatalf("apply the cuit format migration: %v", err)
	}

	want := map[string]sql.NullString{
		"00000000-0000-0000-0000-000000000001": {String: "30712345678", Valid: true},
		"00000000-0000-0000-0000-000000000002": {String: "20123456789", Valid: true},
		"00000000-0000-0000-0000-000000000003": {},
		"00000000-0000-0000-0000-000000000004": {String: "30709876543", Valid: true},
	}
	for id, wantCuit := range want {
		var got sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT cuit FROM inmobiliarias WHERE id = $1::uuid`, id).Scan(&got); err != nil {
			t.Fatalf("read cuit of %s: %v", id, err)
		}
		if got != wantCuit {
			t.Errorf("cuit of %s = %v, want %v", id, got, wantCuit)
		}
	}

	_, err := db.ExecContext(ctx, `INSERT INTO inmobiliarias (razon_social, cuit) VALUES ('Nueva', '30-11111111-1')`)
	assertCheckViolation(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO inmobiliarias (razon_social, cuit) VALUES ('Corta', '123')`)
	assertCheckViolation(t, err)
	_, err = db.ExecContext(ctx, `UPDATE inmobiliarias SET cuit = '30-71234567-9' WHERE id = '00000000-0000-0000-0000-000000000001'`)
	assertCheckViolation(t, err)
	if _, err := db.ExecContext(ctx, `INSERT INTO inmobiliarias (razon_social, cuit) VALUES ('Sin cuit otra vez', NULL)`); err != nil {
		t.Fatalf("insert an inmobiliaria without cuit: %v", err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO inmobiliarias (razon_social, cuit) VALUES ('Duplicada', '30712345678')`)
	assertUniqueViolation(t, err)

	if _, err := provider.DownTo(ctx, 15); err != nil {
		t.Fatalf("roll back the cuit format migration: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO inmobiliarias (razon_social, cuit) VALUES ('Con guiones otra vez', '30-22222222-2')`); err != nil {
		t.Fatalf("insert a cuit with separators after the rollback: %v", err)
	}
}

func TestInmobiliariaCuitFormatMigrationStopsOnAmbiguousData(t *testing.T) {
	cases := []struct {
		name    string
		seed    string
		wantMsg string
	}{
		{
			name: "active duplicate once normalized",
			seed: `INSERT INTO inmobiliarias (razon_social, cuit) VALUES
				('Con guiones', '30-71234567-8'),
				('Sin guiones', '30712345678')`,
			wantMsg: "shared by more than one active row",
		},
		{
			name:    "not 11 digits once normalized",
			seed:    `INSERT INTO inmobiliarias (razon_social, cuit) VALUES ('Libre', 'pendiente')`,
			wantMsg: "not 11 digits",
		},
		{
			name:    "not 11 digits on a row dada de baja",
			seed:    `INSERT INTO inmobiliarias (razon_social, cuit, fecha_baja) VALUES ('Baja', '12', now())`,
			wantMsg: "not 11 digits",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			db, provider := newCuitMigrationProvider(t, ctx)
			if _, err := db.ExecContext(ctx, tc.seed); err != nil {
				t.Fatalf("seed ambiguous inmobiliarias: %v", err)
			}

			_, err := provider.UpTo(ctx, 16)
			if err == nil || !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("UpTo(16) error = %v, want it to mention %q", err, tc.wantMsg)
			}
		})
	}
}
