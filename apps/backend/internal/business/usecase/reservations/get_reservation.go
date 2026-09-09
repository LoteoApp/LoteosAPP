package reservations

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type GetReservation interface {
	Execute(ctx context.Context, actor Actor, id string) (domain.Reservation, error)
}

type getReservationUseCase struct {
	repository gateway.ReservationRepository
}

func NewGetReservation(repository gateway.ReservationRepository) GetReservation {
	return &getReservationUseCase{repository: repository}
}

func (useCase *getReservationUseCase) Execute(ctx context.Context, actor Actor, id string) (domain.Reservation, error) {
	scope, err := reservationScope(actor)
	if err != nil {
		return domain.Reservation{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	reservation, err := useCase.repository.Get(ctx, id, scope)
	if err != nil {
		return domain.Reservation{}, fromRepository(err)
	}
	return reservation, nil
}
