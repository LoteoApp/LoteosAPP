package reservations

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type CancelReservationInput struct {
	Actor         Actor
	ReservationID string
	Reason        string
}

type CancelReservation interface {
	Execute(ctx context.Context, input CancelReservationInput) (domain.Reservation, error)
}

type cancelReservationUseCase struct {
	repository gateway.ReservationRepository
	users      gateway.UserRepository
	clock      Clock
}

func NewCancelReservation(repository gateway.ReservationRepository, users gateway.UserRepository, clocks ...Clock) CancelReservation {
	clock := Clock(SystemClock{})
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &cancelReservationUseCase{repository: repository, users: users, clock: clock}
}

func (useCase *cancelReservationUseCase) Execute(ctx context.Context, input CancelReservationInput) (domain.Reservation, error) {
	if !hasReservationWriteRole(input.Actor.Roles) {
		return domain.Reservation{}, domain.ErrNoAutorizado
	}
	if err := domain.ValidateReservationReason(input.Reason); err != nil {
		return domain.Reservation{}, err
	}
	actor, err := resolveActor(ctx, useCase.users, input.Actor)
	if err != nil {
		return domain.Reservation{}, fromRepository(err)
	}
	if !hasReservationWriteRole([]string{string(actor.Rol)}) {
		return domain.Reservation{}, domain.ErrNoAutorizado
	}
	scope, err := reservationScope(input.Actor)
	if err != nil {
		return domain.Reservation{}, err
	}
	reservationID := strings.TrimSpace(input.ReservationID)
	if reservationID == "" {
		return domain.Reservation{}, domain.ErrReservationNotFound
	}
	reservation, err := useCase.repository.Cancel(ctx, gateway.CancelReservationCommand{
		ReservationID:       reservationID,
		ActorID:             actor.ID,
		ActorAuthProviderID: input.Actor.AuthProviderID,
		Reason:              strings.TrimSpace(input.Reason),
		CancelledAt:         useCase.clock.Now().UTC(),
	}, scope)
	if err != nil {
		return domain.Reservation{}, fromRepository(err)
	}
	return reservation, nil
}
