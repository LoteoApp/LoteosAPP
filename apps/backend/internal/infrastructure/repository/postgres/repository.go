package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (repository *Repository) Snapshot(ctx context.Context) (domain.DatabaseInfo, domain.PoolInfo, error) {
	var database domain.DatabaseInfo
	err := repository.pool.QueryRow(ctx, `
		SELECT
			version(),
			current_database(),
			current_user,
			COALESCE(inet_server_addr()::text, 'local'),
			COALESCE(inet_server_port(), 0),
			current_timestamp
	`).Scan(
		&database.Version,
		&database.DatabaseName,
		&database.User,
		&database.ServerAddress,
		&database.ServerPort,
		&database.DatabaseTime,
	)
	if err != nil {
		return domain.DatabaseInfo{}, domain.PoolInfo{}, err
	}

	stat := repository.pool.Stat()
	pool := domain.PoolInfo{
		MaxConnections:      repository.pool.Config().MaxConns,
		TotalConnections:    stat.TotalConns(),
		AcquiredConnections: stat.AcquiredConns(),
		IdleConnections:     stat.IdleConns(),
		NewConnections:      stat.NewConnsCount(),
		ClosedConnections:   stat.MaxLifetimeDestroyCount() + stat.MaxIdleDestroyCount(),
	}

	return database, pool, nil
}

func (repository *Repository) Ping(ctx context.Context) error {
	return repository.pool.Ping(ctx)
}
