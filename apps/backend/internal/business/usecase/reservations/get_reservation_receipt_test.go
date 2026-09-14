package reservations_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
	"loteosapp/backend/internal/business/usecase/reservations"
)

func TestGetReservationReceiptLoadsAuthorizedReservationAndHistoricalPlan(t *testing.T) {
	issuedAt := time.Date(2026, time.September, 11, 10, 30, 0, 0, time.FixedZone("ART", -3*60*60))
	reservationRepository := &gatewayfake.ReservationRepository{GetResult: domain.Reservation{
		ID: "reservation-1", LoteoID: "authoritative-loteo", LoteID: "lot-7",
	}}
	loteoRepository := &gatewayfake.LoteoRepository{GetResult: domain.Loteo{
		ID: "authoritative-loteo", Name: "Las Acacias",
	}}
	useCase := reservations.NewGetReservationReceipt(reservationRepository, loteoRepository, fixedClock{now: issuedAt})
	actor := reservations.Actor{AuthProviderID: "agency-subject", Roles: []string{domain.RolInmobiliaria}}

	receipt, err := useCase.Execute(context.Background(), actor, " reservation-1 ")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if receipt.Reservation.ID != "reservation-1" || receipt.Loteo.ID != "authoritative-loteo" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if !receipt.IssuedAt.Equal(issuedAt.UTC()) || receipt.IssuedAt.Location() != time.UTC {
		t.Errorf("issued at = %s, want %s in UTC", receipt.IssuedAt, issuedAt.UTC())
	}
	if reservationRepository.GetID != "reservation-1" || reservationRepository.GetScope.AssigneeAuthProviderID == nil || !reservationRepository.GetScope.ByAgencyAssignment {
		t.Errorf("reservation lookup = %q/%#v", reservationRepository.GetID, reservationRepository.GetScope)
	}
	if loteoRepository.GetLoteoID != "authoritative-loteo" {
		t.Errorf("loteo lookup id = %q", loteoRepository.GetLoteoID)
	}
	if loteoRepository.GetScope.AssigneeAuthProviderID != nil || loteoRepository.GetScope.ByUserAssignment || loteoRepository.GetScope.ByAgencyAssignment {
		t.Errorf("historical loteo scope = %#v, want empty scope", loteoRepository.GetScope)
	}
}

func TestGetReservationReceiptAuthorizesBeforeLoadingThePlan(t *testing.T) {
	ctx := context.Background()

	t.Run("unsupported role", func(t *testing.T) {
		reservationsRepository := &gatewayfake.ReservationRepository{}
		loteosRepository := &gatewayfake.LoteoRepository{}
		useCase := reservations.NewGetReservationReceipt(reservationsRepository, loteosRepository)

		_, err := useCase.Execute(ctx, reservations.Actor{AuthProviderID: "subject", Roles: []string{domain.RolAgrimensor}}, "reservation-1")
		if !errors.Is(err, domain.ErrNoAutorizado) {
			t.Fatalf("Execute() error = %v", err)
		}
		if reservationsRepository.GetCalls != 0 || loteosRepository.GetCalls != 0 {
			t.Fatalf("repository calls = reservation %d, loteo %d", reservationsRepository.GetCalls, loteosRepository.GetCalls)
		}
	})

	t.Run("missing reservation", func(t *testing.T) {
		reservationsRepository := &gatewayfake.ReservationRepository{GetErr: domain.ErrReservationNotFound}
		loteosRepository := &gatewayfake.LoteoRepository{}
		useCase := reservations.NewGetReservationReceipt(reservationsRepository, loteosRepository)

		_, err := useCase.Execute(ctx, reservations.Actor{AuthProviderID: "subject", Roles: []string{domain.RolAdministrador}}, "reservation-1")
		if !errors.Is(err, domain.ErrReservationNotFound) {
			t.Fatalf("Execute() error = %v", err)
		}
		if loteosRepository.GetCalls != 0 {
			t.Fatalf("loteo loaded before reservation authorization: %d calls", loteosRepository.GetCalls)
		}
	})

	t.Run("empty id", func(t *testing.T) {
		reservationsRepository := &gatewayfake.ReservationRepository{}
		loteosRepository := &gatewayfake.LoteoRepository{}
		useCase := reservations.NewGetReservationReceipt(reservationsRepository, loteosRepository)

		_, err := useCase.Execute(ctx, reservations.Actor{AuthProviderID: "subject", Roles: []string{domain.RolAdministrador}}, " ")
		if !errors.Is(err, domain.ErrReservationNotFound) || reservationsRepository.GetCalls != 0 || loteosRepository.GetCalls != 0 {
			t.Fatalf("error/calls = %v/%d/%d", err, reservationsRepository.GetCalls, loteosRepository.GetCalls)
		}
	})
}

func TestGetReservationReceiptMapsRepositoryFailures(t *testing.T) {
	ctx := context.Background()
	actor := reservations.Actor{AuthProviderID: "subject", Roles: []string{domain.RolAdministrativo}}

	t.Run("reservation", func(t *testing.T) {
		cause := errors.New("reservation lookup failed")
		useCase := reservations.NewGetReservationReceipt(
			&gatewayfake.ReservationRepository{GetErr: cause},
			&gatewayfake.LoteoRepository{},
		)
		_, err := useCase.Execute(ctx, actor, "reservation-1")
		if !errors.Is(err, domain.ErrDatabaseUnavailable) || !errors.Is(err, cause) {
			t.Fatalf("Execute() error = %v", err)
		}
	})

	t.Run("loteo", func(t *testing.T) {
		cause := errors.New("loteo lookup failed")
		useCase := reservations.NewGetReservationReceipt(
			&gatewayfake.ReservationRepository{GetResult: domain.Reservation{LoteoID: "loteo-1"}},
			&gatewayfake.LoteoRepository{GetErr: cause},
		)
		_, err := useCase.Execute(ctx, actor, "reservation-1")
		if !errors.Is(err, domain.ErrDatabaseUnavailable) || !errors.Is(err, cause) {
			t.Fatalf("Execute() error = %v", err)
		}
	})

	t.Run("loteo domain error", func(t *testing.T) {
		useCase := reservations.NewGetReservationReceipt(
			&gatewayfake.ReservationRepository{GetResult: domain.Reservation{LoteoID: "loteo-1"}},
			&gatewayfake.LoteoRepository{GetErr: domain.ErrLoteoNotFound},
		)
		_, err := useCase.Execute(ctx, actor, "reservation-1")
		if !errors.Is(err, domain.ErrLoteoNotFound) {
			t.Fatalf("Execute() error = %v", err)
		}
	})
}
