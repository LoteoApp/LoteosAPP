package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"loteosapp/backend/internal/app"
)

// healthcheckCommand is the CLI subcommand the container runtime invokes to
// probe liveness. The distroless runtime image ships no shell and no
// wget/curl, so the healthcheck runs through the server binary itself
// instead of an external tool (see docs/deployment.md).
const healthcheckCommand = "healthcheck"

func main() {
	if len(os.Args) > 1 && os.Args[1] == healthcheckCommand {
		os.Exit(runHealthcheck())
	}

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

// runHealthcheck reports whether the local server answers GET /health with
// 200 OK. It reads PORT directly instead of environments.LoadServer, so it
// doesn't require DATABASE_URL/SUPABASE_*/R2 credentials to run.
func runHealthcheck() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		slog.Error("healthcheck request failed", "error", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("healthcheck failed", "status", resp.StatusCode)
		return 1
	}

	return 0
}
