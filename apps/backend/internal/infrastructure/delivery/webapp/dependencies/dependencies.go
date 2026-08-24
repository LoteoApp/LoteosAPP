package dependencies

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/usecase/system"
	"loteosapp/backend/internal/business/usecase/users"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/environments"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

type Container struct {
	GetSystemInfoHandler          *handler.GetSystemInfoHandler
	CheckDatabaseReadinessHandler *handler.CheckDatabaseReadinessHandler
	CreateUserHandler             *handler.CreateUserHandler
	CompleteProfileHandler        *handler.CompleteProfileHandler
	Pool                          *pgxpool.Pool
	Verifier                      *supabase.Verifier
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
	getSystemInfoHandler := handler.NewGetSystemInfoHandler(system.NewGetSystemInfo(repo))
	checkDatabaseReadinessHandler := handler.NewCheckDatabaseReadinessHandler(system.NewCheckDatabaseReadiness(repo))

	adminClient := supabase.NewAdminClient(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)
	userRepo := postgres.NewUserRepository(pool)
	createUserHandler := handler.NewCreateUserHandler(users.NewCreateUser(userRepo, adminClient))
	completeProfileHandler := handler.NewCompleteProfileHandler(users.NewCompleteProfile(userRepo))

	return &Container{
		GetSystemInfoHandler:          getSystemInfoHandler,
		CheckDatabaseReadinessHandler: checkDatabaseReadinessHandler,
		CreateUserHandler:             createUserHandler,
		CompleteProfileHandler:        completeProfileHandler,
		Pool:                          pool,
		Verifier:                      verifier,
	}, nil
}
