package usecase

import (
	"context"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type Service struct {
	repository gateway.Repository
}

func NewService(repository gateway.Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Info(ctx context.Context) (domain.Info, error) {
	database, pool, err := service.repository.Snapshot(ctx)
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

func (service *Service) Ready(ctx context.Context) error {
	return service.repository.Ping(ctx)
}
