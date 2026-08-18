package dependencies

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/usecase"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/environments"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

type Container struct {
	Handler     *handler.Handler
	UserHandler *handler.UserHandler
	Pool        *pgxpool.Pool
	Verifier    *supabase.Verifier
}

func New(ctx context.Context, cfg environments.Server) (*Container, error) {
	pool, err := postgres.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	verifier, err := supabase.NewVerifier(ctx, cfg.SupabaseURL)
	if err != nil {
		pool.Close()
		return nil, err
	}

	repo := postgres.NewRepository(pool)
	service := usecase.NewService(repo)
	h := handler.NewHandler(service)

	adminClient := supabase.NewAdminClient(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)
	userRepo := postgres.NewUserRepository(pool)
	userService := usecase.NewUserService(userRepo, adminClient)
	uh := handler.NewUserHandler(userService)

	return &Container{
		Handler:     h,
		UserHandler: uh,
		Pool:        pool,
		Verifier:    verifier,
	}, nil
}
