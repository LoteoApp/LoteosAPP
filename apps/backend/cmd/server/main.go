package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"loteosapp/backend/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	startupCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	application, err := app.New(startupCtx)
	if err != nil {
		slog.Error("backend startup failed", "error", err)
		os.Exit(1)
	}

	if err := application.Run(ctx); err != nil {
		slog.Error("backend stopped unexpectedly", "error", err)
		os.Exit(1)
	}
}
