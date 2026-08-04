package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

type Repository interface {
	Snapshot(ctx context.Context) (domain.DatabaseInfo, domain.PoolInfo, error)
	Ping(ctx context.Context) error
}
