package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"loteosapp/backend/internal/infrastructure/environments"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cfg := environments.LoadMigration()

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

	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("migration dialect setup failed", "error", err)
		os.Exit(1)
	}

	if err := goose.UpContext(ctx, db, cfg.MigrationsDir); err != nil {
		slog.Error("database migrations failed", "error", err)
		os.Exit(1)
	}

	slog.Info("database migrations applied")
}
