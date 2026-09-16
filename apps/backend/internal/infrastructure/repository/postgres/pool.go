package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func poolConfig(connectionString string) (*pgxpool.Config, error) {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, err
	}

	config.MaxConns = 4
	config.MinConns = 0
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckPeriod = time.Minute
	config.PingTimeout = 5 * time.Second
	config.ConnConfig.ConnectTimeout = 5 * time.Second

	return config, nil
}

func OpenPool(ctx context.Context, connectionString string) (*pgxpool.Pool, error) {
	config, err := poolConfig(connectionString)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
