package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"loteosapp/backend/internal/infrastructure/environments"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cfg, err := environments.LoadMigration()
	if err != nil {
		slog.Error("migration configuration failed", "error", err)
		os.Exit(1)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		slog.Error("migration database setup failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		slog.Error("migration database connection failed", "error", err)
		os.Exit(1)
	}

	// Waiting for the lock must not consume the whole context budget, or the
	// winner of a race starts migrating with no time left to finish.
	locker, err := lock.NewPostgresSessionLocker(lock.WithLockTimeout(5, 12))
	if err != nil {
		slog.Error("migration locker setup failed", "error", err)
		os.Exit(1)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(cfg.MigrationsDir), goose.WithSessionLocker(locker))
	if err != nil {
		slog.Error("migration provider setup failed", "error", err)
		os.Exit(1)
	}

	if _, err := provider.Up(ctx); err != nil {
		slog.Error("database migrations failed", "error", err)
		os.Exit(1)
	}

	slog.Info("database migrations applied")
}
