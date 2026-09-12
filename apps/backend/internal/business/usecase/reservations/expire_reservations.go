package reservations

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type ProcessExpirations interface {
	Execute(ctx context.Context, limit int) (domain.ExpirationReport, error)
}

type processExpirationsUseCase struct {
	repository gateway.ReservationRepository
	clock      Clock
}

func NewProcessExpirations(repository gateway.ReservationRepository, clocks ...Clock) ProcessExpirations {
	clock := Clock(SystemClock{})
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &processExpirationsUseCase{repository: repository, clock: clock}
}

func (useCase *processExpirationsUseCase) Execute(ctx context.Context, limit int) (domain.ExpirationReport, error) {
	if limit < 1 {
		limit = domain.DefaultReservationLimit
	}
	report, err := useCase.repository.ExpireDue(ctx, useCase.clock.Now().UTC(), limit)
	if err != nil {
		return domain.ExpirationReport{}, fromRepository(err)
	}
	return report, nil
}
