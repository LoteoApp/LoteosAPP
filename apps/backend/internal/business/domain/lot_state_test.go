package domain_test

import (
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
)

func TestLotStateTransitions(t *testing.T) {
	t.Parallel()

	allowed := map[domain.LotState][]domain.LotState{
		domain.LotStateAvailable: {domain.LotStateReserved, domain.LotStateSold},
		domain.LotStateReserved:  {domain.LotStateAvailable, domain.LotStateSold},
		domain.LotStateSold:      {domain.LotStateAvailable, domain.LotStateCompleted},
	}
	all := []domain.LotState{
		domain.LotStateAvailable,
		domain.LotStateReserved,
		domain.LotStateSold,
		domain.LotStateCompleted,
		"unknown",
	}

	for _, current := range all {
		for _, next := range all {
			want := containsState(allowed[current], next)
			if got := current.CanTransitionTo(next); got != want {
				t.Errorf("%q.CanTransitionTo(%q) = %v, want %v", current, next, got, want)
			}
		}
	}
}

func TestLotStateValidation(t *testing.T) {
	t.Parallel()

	for _, state := range []domain.LotState{
		domain.LotStateAvailable,
		domain.LotStateReserved,
		domain.LotStateSold,
		domain.LotStateCompleted,
	} {
		if !state.IsValid() {
			t.Errorf("%q.IsValid() = false", state)
		}
	}
	if domain.LotState("unknown").IsValid() {
		t.Error("an unknown state should be invalid")
	}
}

func TestLotStateTransitionValidation(t *testing.T) {
	t.Parallel()

	reservationID := "reservation-1"
	saleID := "sale-1"
	valid := domain.LotStateTransition{
		ExpectedState: domain.LotStateAvailable,
		NextState:     domain.LotStateReserved,
		Origin:        domain.LotStateOriginReservation,
		ReservationID: &reservationID,
	}

	tests := map[string]struct {
		mutate func(*domain.LotStateTransition)
		want   error
	}{
		"valid reservation": {mutate: func(*domain.LotStateTransition) {}},
		"unknown current state": {
			mutate: func(command *domain.LotStateTransition) { command.ExpectedState = "unknown" },
			want:   domain.ErrInvalidLotState,
		},
		"unknown next state": {
			mutate: func(command *domain.LotStateTransition) { command.NextState = "unknown" },
			want:   domain.ErrInvalidLotState,
		},
		"unknown origin": {
			mutate: func(command *domain.LotStateTransition) { command.Origin = "unknown" },
			want:   domain.ErrInvalidLotStateOrigin,
		},
		"creation origin": {
			mutate: func(command *domain.LotStateTransition) { command.Origin = domain.LotStateOriginCreation },
			want:   domain.ErrInvalidLotStateOrigin,
		},
		"forbidden transition": {
			mutate: func(command *domain.LotStateTransition) { command.NextState = domain.LotStateCompleted },
			want:   domain.ErrInvalidLotStateTransition,
		},
		"reservation without reference": {
			mutate: func(command *domain.LotStateTransition) { command.ReservationID = nil },
			want:   domain.ErrLotStateReferenceRequired,
		},
		"sale without reference": {
			mutate: func(command *domain.LotStateTransition) {
				command.Origin = domain.LotStateOriginSale
				command.NextState = domain.LotStateSold
				command.ReservationID = nil
			},
			want: domain.ErrLotStateReferenceRequired,
		},
		"sale with reference": {
			mutate: func(command *domain.LotStateTransition) {
				command.Origin = domain.LotStateOriginSale
				command.NextState = domain.LotStateSold
				command.ReservationID = nil
				command.SaleID = &saleID
			},
		},
		"manual reservation cancellation without reason": {
			mutate: func(command *domain.LotStateTransition) {
				command.ExpectedState = domain.LotStateReserved
				command.NextState = domain.LotStateAvailable
			},
			want: domain.ErrLotStateReasonRequired,
		},
		"manual reservation cancellation with reason": {
			mutate: func(command *domain.LotStateTransition) {
				command.ExpectedState = domain.LotStateReserved
				command.NextState = domain.LotStateAvailable
				command.Reason = "El cliente desistio"
			},
		},
		"collection cannot cancel a sale": {
			mutate: func(command *domain.LotStateTransition) {
				command.ExpectedState = domain.LotStateSold
				command.NextState = domain.LotStateAvailable
				command.Origin = domain.LotStateOriginCollection
				command.ReservationID = nil
				command.SaleID = &saleID
			},
			want: domain.ErrInvalidLotStateOrigin,
		},
		"collection completes a sale": {
			mutate: func(command *domain.LotStateTransition) {
				command.ExpectedState = domain.LotStateSold
				command.NextState = domain.LotStateCompleted
				command.Origin = domain.LotStateOriginCollection
				command.ReservationID = nil
				command.SaleID = &saleID
			},
		},
		"system releases an expired reservation": {
			mutate: func(command *domain.LotStateTransition) {
				command.ExpectedState = domain.LotStateReserved
				command.NextState = domain.LotStateAvailable
				command.Origin = domain.LotStateOriginSystem
				command.ReservationID = nil
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			command := valid
			test.mutate(&command)

			err := command.Validate()
			if !errors.Is(err, test.want) {
				t.Fatalf("Validate() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestLotStateTransitionOrigins(t *testing.T) {
	t.Parallel()

	reservationID := "reservation-1"
	saleID := "sale-1"
	tests := []domain.LotStateTransition{
		{ExpectedState: domain.LotStateAvailable, NextState: domain.LotStateReserved, Origin: domain.LotStateOriginReservation, ReservationID: &reservationID},
		{ExpectedState: domain.LotStateAvailable, NextState: domain.LotStateSold, Origin: domain.LotStateOriginSale, SaleID: &saleID},
		{ExpectedState: domain.LotStateReserved, NextState: domain.LotStateAvailable, Origin: domain.LotStateOriginReservation, Reason: "cancelled", ReservationID: &reservationID},
		{ExpectedState: domain.LotStateReserved, NextState: domain.LotStateAvailable, Origin: domain.LotStateOriginSystem},
		{ExpectedState: domain.LotStateReserved, NextState: domain.LotStateSold, Origin: domain.LotStateOriginSale, SaleID: &saleID},
		{ExpectedState: domain.LotStateSold, NextState: domain.LotStateAvailable, Origin: domain.LotStateOriginSale, Reason: "cancelled", SaleID: &saleID},
		{ExpectedState: domain.LotStateSold, NextState: domain.LotStateAvailable, Origin: domain.LotStateOriginSystem},
		{ExpectedState: domain.LotStateSold, NextState: domain.LotStateCompleted, Origin: domain.LotStateOriginCollection, SaleID: &saleID},
	}

	for _, command := range tests {
		if err := command.Validate(); err != nil {
			t.Errorf("Validate(%q -> %q, %q) error = %v", command.ExpectedState, command.NextState, command.Origin, err)
		}
	}
}

func containsState(states []domain.LotState, wanted domain.LotState) bool {
	for _, state := range states {
		if state == wanted {
			return true
		}
	}
	return false
}
