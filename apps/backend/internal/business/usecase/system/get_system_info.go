package system

import (
	"context"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// GetSystemInfo reports a diagnostic snapshot of the database and connection
// pool, served at GET /api/v1/system.
type GetSystemInfo interface {
	Execute(ctx context.Context) (domain.Info, error)
}

type getSystemInfoUseCase struct {
	repository gateway.Repository
}

func NewGetSystemInfo(repository gateway.Repository) GetSystemInfo {
	return &getSystemInfoUseCase{repository: repository}
}

func (useCase *getSystemInfoUseCase) Execute(ctx context.Context) (domain.Info, error) {
	database, pool, err := useCase.repository.Snapshot(ctx)
	if err != nil {
		return domain.Info{}, err
	}

	database.Connected = true

	return domain.Info{
		Service:   "loteosapp-backend",
		Status:    "ok",
		CheckedAt: time.Now().UTC(),
		Database:  database,
		Pool:      pool,
	}, nil
}
