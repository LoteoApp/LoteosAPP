package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/infrastructure/auth/keycloak"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/dependencies"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/route"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/server"
	"loteosapp/backend/internal/infrastructure/environments"
)

type App struct {
	server   *http.Server
	pool     *pgxpool.Pool
	verifier *keycloak.Verifier
}

func New(ctx context.Context) (*App, error) {
	cfg := environments.LoadServer()

	container, err := dependencies.New(ctx, cfg.DatabaseURL, cfg.KeycloakIssuer, cfg.KeycloakAudience)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	route.RegisterRoutes(mux, container.Handler)

	return &App{
		server:   server.New(cfg.Port, server.WithCORS(cfg.FrontendOrigin, mux)),
		pool:     container.Pool,
		verifier: container.Verifier,
	}, nil
}

func (app *App) Run(ctx context.Context) error {
	defer app.pool.Close()
	defer app.verifier.Close()

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.server.ListenAndServe()
	}()

	slog.Info("backend listening", "address", app.server.Addr)

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := app.server.Shutdown(shutdownCtx); err != nil {
			return err
		}

		err := <-serverErrors
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
