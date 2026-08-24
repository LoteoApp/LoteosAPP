package gatewayfake

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

// Repository is a fake gateway.Repository for tests.
type Repository struct {
	Database    domain.DatabaseInfo
	Pool        domain.PoolInfo
	SnapshotErr error
	PingErr     error
}

func (repository Repository) Snapshot(context.Context) (domain.DatabaseInfo, domain.PoolInfo, error) {
	return repository.Database, repository.Pool, repository.SnapshotErr
}

func (repository Repository) Ping(context.Context) error {
	return repository.PingErr
}
