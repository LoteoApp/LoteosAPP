package dependencies

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/usecase"
	"loteosapp/backend/internal/infrastructure/auth/keycloak"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/handler"
	"loteosapp/backend/internal/infrastructure/repository/postgres"
)

type Container struct {
	Handler  *handler.Handler
	Pool     *pgxpool.Pool
	Verifier *keycloak.Verifier
}

func New(ctx context.Context, databaseURL, keycloakIssuer, keycloakAudience string) (*Container, error) {
	pool, err := postgres.OpenPool(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	verifier, err := keycloak.NewVerifier(ctx, keycloakIssuer, keycloakAudience)
	if err != nil {
		pool.Close()
		return nil, err
	}

	repo := postgres.NewRepository(pool)
	service := usecase.NewService(repo)
	h := handler.NewHandler(service)

	return &Container{
		Handler:  h,
		Pool:     pool,
		Verifier: verifier,
	}, nil
}
