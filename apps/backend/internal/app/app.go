package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/dependencies"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/route"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/server"
	"loteosapp/backend/internal/infrastructure/environments"
)

type App struct {
	server   *http.Server
	pool     *pgxpool.Pool
	verifier *supabase.Verifier
}

func New(ctx context.Context) (*App, error) {
	cfg, err := environments.LoadServer()
	if err != nil {
		return nil, err
	}

	container, err := dependencies.New(ctx, cfg)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	route.RegisterRoutes(mux, route.Handlers{
		CreateUser:      container.CreateUserHandler,
		CompleteProfile: container.CompleteProfileHandler,
		ListUsers:       container.ListUsersHandler,
		UpdateUser:      container.UpdateUserHandler,
		DeactivateUser:  container.DeactivateUserHandler,
		ReactivateUser:  container.ReactivateUserHandler,
		CreateClient:    container.CreateClientHandler,
		UpdateClient:    container.UpdateClientHandler,
		DeleteClient:    container.DeleteClientHandler,
		ListClients:     container.ListClientsHandler,
		CreateAgency:    container.CreateAgencyHandler,
		UpdateAgency:    container.UpdateAgencyHandler,
		DeleteAgency:    container.DeleteAgencyHandler,
		ListAgencies:    container.ListAgenciesHandler,
		CreateLoteo:     container.CreateLoteoHandler,
		StoreLoteoDxf:   container.StoreLoteoDxfHandler,
		UpdateLote:      container.UpdateLoteHandler,
		UpdateManzana:   container.UpdateManzanaHandler,
		UpdateCalle:     container.UpdateCalleHandler,
		ListLoteos:      container.ListLoteosHandler,
		GetLoteo:        container.GetLoteoHandler,
	}, container.Verifier, container.UserRepository)

	return &App{
		server:   server.New(cfg.Port, server.WithCORS(cfg.FrontendOrigin, mux), route.MaxHandlerTimeout),
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
