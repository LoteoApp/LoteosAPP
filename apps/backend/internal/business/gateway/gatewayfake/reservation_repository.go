package gatewayfake

import (
	"context"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type ReservationRepository struct {
	CreateCalls   int
	CreateErr     error
	CreateResult  domain.Reservation
	CreateCommand gateway.CreateReservationCommand

	ListCalls  int
	ListErr    error
	ListResult domain.ReservationPage
	ListFilter domain.ReservationListFilter
	ListScope  gateway.ReservationScope

	GetCalls  int
	GetErr    error
	GetResult domain.Reservation
	GetID     string
	GetScope  gateway.ReservationScope

	CancelCalls   int
	CancelErr     error
	CancelResult  domain.Reservation
	CancelCommand gateway.CancelReservationCommand
	CancelScope   gateway.ReservationScope

	SellersCalls  int
	SellersErr    error
	SellersResult []domain.SellerOption
	SellersLoteo  string
	SellersScope  gateway.ReservationScope

	ExpireCalls  int
	ExpireErr    error
	ExpireResult domain.ExpirationReport
	ExpireNow    time.Time
	ExpireLimit  int
}

func (fake *ReservationRepository) Create(_ context.Context, command gateway.CreateReservationCommand) (domain.Reservation, error) {
	fake.CreateCalls++
	fake.CreateCommand = command
	if fake.CreateErr != nil {
		return domain.Reservation{}, fake.CreateErr
	}
	return fake.CreateResult, nil
}

func (fake *ReservationRepository) List(_ context.Context, filter domain.ReservationListFilter, scope gateway.ReservationScope) (domain.ReservationPage, error) {
	fake.ListCalls++
	fake.ListFilter = filter
	fake.ListScope = scope
	if fake.ListErr != nil {
		return domain.ReservationPage{}, fake.ListErr
	}
	return fake.ListResult, nil
}

func (fake *ReservationRepository) Get(_ context.Context, id string, scope gateway.ReservationScope) (domain.Reservation, error) {
	fake.GetCalls++
	fake.GetID = id
	fake.GetScope = scope
	if fake.GetErr != nil {
		return domain.Reservation{}, fake.GetErr
	}
	return fake.GetResult, nil
}

func (fake *ReservationRepository) Cancel(_ context.Context, command gateway.CancelReservationCommand, scope gateway.ReservationScope) (domain.Reservation, error) {
	fake.CancelCalls++
	fake.CancelCommand = command
	fake.CancelScope = scope
	if fake.CancelErr != nil {
		return domain.Reservation{}, fake.CancelErr
	}
	return fake.CancelResult, nil
}

func (fake *ReservationRepository) ListEligibleSellers(_ context.Context, loteoID string, scope gateway.ReservationScope) ([]domain.SellerOption, error) {
	fake.SellersCalls++
	fake.SellersLoteo = loteoID
	fake.SellersScope = scope
	if fake.SellersErr != nil {
		return nil, fake.SellersErr
	}
	return fake.SellersResult, nil
}

func (fake *ReservationRepository) ExpireDue(_ context.Context, now time.Time, limit int) (domain.ExpirationReport, error) {
	fake.ExpireCalls++
	fake.ExpireNow = now
	fake.ExpireLimit = limit
	if fake.ExpireErr != nil {
		return domain.ExpirationReport{}, fake.ExpireErr
	}
	return fake.ExpireResult, nil
}
