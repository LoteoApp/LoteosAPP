package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/platform/config"
	"loteosapp/backend/internal/platform/httpserver"
	platformpostgres "loteosapp/backend/internal/platform/postgres"
	"loteosapp/backend/internal/system"
	systemhttp "loteosapp/backend/internal/system/http"
	systempostgres "loteosapp/backend/internal/system/postgres"
)

type App struct {
	server *http.Server
	pool   *pgxpool.Pool
}

func New(ctx context.Context) (*App, error) {
	cfg := config.LoadServer()

	pool, err := platformpostgres.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	repository := systempostgres.NewRepository(pool)
	service := system.NewService(repository)
	handler := systemhttp.NewHandler(service)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return &App{
		server: httpserver.New(cfg.Port, httpserver.WithCORS(cfg.FrontendOrigin, mux)),
		pool:   pool,
	}, nil
}

func (app *App) Run(ctx context.Context) error {
	defer app.pool.Close()

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
