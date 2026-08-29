package dependencies

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/gateway"
	"loteosapp/backend/internal/business/usecase/users"
	"loteosapp/backend/internal/infrastructure/auth/supabase"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/environments"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
	"loteosapp/backend/internal/infrastructure/storage/r2"
)

type Container struct {
	CreateUserHandler      *handler.CreateUserHandler
	CompleteProfileHandler *handler.CompleteProfileHandler
	Pool                   *pgxpool.Pool
	Verifier               *supabase.Verifier
	ObjectStorage          gateway.ObjectStorage
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

	objectStorage, err := r2.NewClient(r2.Config{
		Endpoint:        cfg.Storage.Endpoint,
		Bucket:          cfg.Storage.Bucket,
		AccessKeyID:     cfg.Storage.AccessKeyID,
		SecretAccessKey: cfg.Storage.SecretAccessKey,
	})
	if err != nil {
		pool.Close()
		return nil, err
	}

	adminClient := supabase.NewAdminClient(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)
	userRepo := postgres.NewUserRepository(pool)
	createUserHandler := handler.NewCreateUserHandler(users.NewCreateUser(userRepo, adminClient))
	completeProfileHandler := handler.NewCompleteProfileHandler(users.NewCompleteProfile(userRepo))

	return &Container{
		CreateUserHandler:      createUserHandler,
		CompleteProfileHandler: completeProfileHandler,
		Pool:                   pool,
		Verifier:               verifier,
		ObjectStorage:          objectStorage,
	}, nil
}
