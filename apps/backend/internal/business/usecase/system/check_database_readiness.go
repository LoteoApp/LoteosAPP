package system

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// CheckDatabaseReadiness checks that the database is reachable, served at
// GET /readyz.
type CheckDatabaseReadiness interface {
	Execute(ctx context.Context) error
}

type checkDatabaseReadinessUseCase struct {
	repository gateway.Repository
}

func NewCheckDatabaseReadiness(repository gateway.Repository) CheckDatabaseReadiness {
	return &checkDatabaseReadinessUseCase{repository: repository}
}

func (useCase *checkDatabaseReadinessUseCase) Execute(ctx context.Context) error {
	if err := useCase.repository.Ping(ctx); err != nil {
		return &domain.Error{
			Kind:    domain.KindUnavailable,
			Code:    "database_unavailable",
			Message: "La base de datos no está disponible",
			Cause:   err,
		}
	}
	return nil
}
