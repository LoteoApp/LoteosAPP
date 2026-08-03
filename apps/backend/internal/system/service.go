package system

import (
	"context"
	"time"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (service *Service) Info(ctx context.Context) (Info, error) {
	database, pool, err := service.repository.Snapshot(ctx)
	if err != nil {
		return Info{}, err
	}

	database.Connected = true

	return Info{
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
