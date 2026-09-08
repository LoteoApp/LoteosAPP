package domain_test

import (
	"errors"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
)

func TestReservationStateTransitions(t *testing.T) {
	for _, test := range []struct {
		from domain.ReservationState
		to   domain.ReservationState
		want bool
	}{
		{domain.ReservationStateActive, domain.ReservationStateExpired, true},
		{domain.ReservationStateActive, domain.ReservationStateCancelled, true},
		{domain.ReservationStateActive, domain.ReservationStateConverted, true},
		{domain.ReservationStateExpired, domain.ReservationStateActive, false},
		{domain.ReservationStateCancelled, domain.ReservationStateExpired, false},
	} {
		if got := test.from.CanTransitionTo(test.to); got != test.want {
			t.Errorf("%q -> %q = %v, want %v", test.from, test.to, got, test.want)
		}
	}
}

func TestReservationAuthorizationRules(t *testing.T) {
	if !domain.IsReservationRole(domain.RolAdministrador) || !domain.IsReservationRole(domain.RolInmobiliaria) {
		t.Fatal("reservation roles should be accepted")
	}
	if domain.IsReservationRole(domain.RolAgrimensor) || domain.IsAdministrativeRole(domain.RolInmobiliaria) {
		t.Fatal("non-reservation roles should be rejected")
	}
	cases := []struct {
		name           string
		role           domain.Rol
		actorID        string
		sellerID       string
		sameAgency     bool
		agencyAssigned bool
		want           bool
	}{
		{"administrator", domain.RolAdministrador, "a", "s", false, false, true},
		{"seller", domain.RolInmobiliaria, "same", "same", false, false, true},
		{"assigned agency", domain.RolInmobiliaria, "a", "s", true, true, true},
		{"unassigned agency", domain.RolInmobiliaria, "a", "s", true, false, false},
		{"other agency", domain.RolInmobiliaria, "a", "s", false, true, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := domain.CanCancelReservation(test.role, test.actorID, test.sellerID, test.sameAgency, test.agencyAssigned); got != test.want {
				t.Errorf("CanCancelReservation() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestValidateReservationReason(t *testing.T) {
	if !errors.Is(domain.ValidateReservationReason("  "), domain.ErrReservationReasonRequired) {
		t.Error("blank reason should be rejected")
	}
	if !errors.Is(domain.ValidateReservationReason(string(make([]rune, domain.MaxReservationReasonSize+1))), domain.ErrReservationReasonTooLong) {
		t.Error("long reason should be rejected")
	}
	if err := domain.ValidateReservationReason("Cliente desistió"); err != nil {
		t.Fatalf("valid reason returned %v", err)
	}
}

func TestReservationDurationAndBoundary(t *testing.T) {
	created := time.Date(2026, time.January, 31, 20, 0, 0, 0, time.UTC)
	expires := created.Add(domain.ReservationDuration)
	if expires.Sub(created) != 360*time.Hour {
		t.Fatalf("duration = %s", expires.Sub(created))
	}
	if created.Before(expires) && !expires.Equal(created.Add(360*time.Hour)) {
		t.Fatal("expiration should be exactly 360 hours")
	}
}

func TestReservationListFilterNormalize(t *testing.T) {
	filter, err := (domain.ReservationListFilter{States: []domain.ReservationState{domain.ReservationStateActive}}).Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if filter.Page != 1 || filter.Limit != 25 {
		t.Fatalf("defaults = %#v", filter)
	}
	if _, err := (domain.ReservationListFilter{Page: 0, Limit: 101}).Normalize(); !errors.Is(err, domain.ErrReservationInvalidPage) {
		t.Fatalf("invalid limit error = %v", err)
	}
	if _, err := (domain.ReservationListFilter{States: []domain.ReservationState{"unknown"}}).Normalize(); !errors.Is(err, domain.ErrReservationInvalidState) {
		t.Fatalf("invalid state error = %v", err)
	}
}
