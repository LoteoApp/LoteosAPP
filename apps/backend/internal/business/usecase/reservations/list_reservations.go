package reservations

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type ListReservationsInput struct {
	Actor  Actor
	Filter domain.ReservationListFilter
}

type ListReservations interface {
	Execute(ctx context.Context, input ListReservationsInput) (domain.ReservationPage, error)
}

type listReservationsUseCase struct {
	repository gateway.ReservationRepository
}

func NewListReservations(repository gateway.ReservationRepository) ListReservations {
	return &listReservationsUseCase{repository: repository}
}

func (useCase *listReservationsUseCase) Execute(ctx context.Context, input ListReservationsInput) (domain.ReservationPage, error) {
	scope, err := reservationScope(input.Actor)
	if err != nil {
		return domain.ReservationPage{}, err
	}
	filter, err := input.Filter.Normalize()
	if err != nil {
		return domain.ReservationPage{}, err
	}
	page, err := useCase.repository.List(ctx, filter, scope)
	if err != nil {
		return domain.ReservationPage{}, fromRepository(err)
	}
	return page, nil
}
