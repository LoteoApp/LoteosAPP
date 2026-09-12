package reservations

import (
	"context"
	"strings"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type ReservationReceipt struct {
	Reservation domain.Reservation
	Loteo       domain.Loteo
	IssuedAt    time.Time
}

type GetReservationReceipt interface {
	Execute(ctx context.Context, actor Actor, id string) (ReservationReceipt, error)
}

type getReservationReceiptUseCase struct {
	reservations gateway.ReservationRepository
	loteos       gateway.LoteoRepository
	clock        Clock
}

func NewGetReservationReceipt(
	reservations gateway.ReservationRepository,
	loteos gateway.LoteoRepository,
	clocks ...Clock,
) GetReservationReceipt {
	clock := Clock(SystemClock{})
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &getReservationReceiptUseCase{reservations: reservations, loteos: loteos, clock: clock}
}

func (useCase *getReservationReceiptUseCase) Execute(ctx context.Context, actor Actor, id string) (ReservationReceipt, error) {
	scope, err := reservationScope(actor)
	if err != nil {
		return ReservationReceipt{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return ReservationReceipt{}, domain.ErrReservationNotFound
	}

	reservation, err := useCase.reservations.Get(ctx, id, scope)
	if err != nil {
		return ReservationReceipt{}, fromRepository(err)
	}
	loteo, err := useCase.loteos.Get(ctx, reservation.LoteoID, gateway.LoteoScope{})
	if err != nil {
		return ReservationReceipt{}, fromRepository(err)
	}

	return ReservationReceipt{
		Reservation: reservation,
		Loteo:       loteo,
		IssuedAt:    useCase.clock.Now().UTC(),
	}, nil
}
