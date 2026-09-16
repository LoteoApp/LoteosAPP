package domain_test

import (
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
)

func TestSaleStateAndPaymentMethodValidity(t *testing.T) {
	for _, state := range []domain.SaleState{domain.SaleStateActive, domain.SaleStateCompleted, domain.SaleStateCancelled} {
		if !state.IsValid() {
			t.Errorf("%q should be a valid sale state", state)
		}
	}
	if domain.SaleState("vencida").IsValid() {
		t.Error("vencida is not a sale state")
	}
	for _, method := range []domain.PaymentMethod{domain.PaymentMethodCash, domain.PaymentMethodFinanced, domain.PaymentMethodDownAndFi} {
		if !method.IsValid() {
			t.Errorf("%q should be a valid payment method", method)
		}
	}
	if domain.PaymentMethod("cripto").IsValid() {
		t.Error("cripto is not a payment method")
	}
	if !domain.PaymentMethodCash.IsAvailable() || domain.PaymentMethodFinanced.IsAvailable() || domain.PaymentMethodDownAndFi.IsAvailable() {
		t.Error("only contado should be available")
	}
	if !domain.PaymentMethodCash.SettlesOnRegistration() || domain.PaymentMethodFinanced.SettlesOnRegistration() || domain.PaymentMethodDownAndFi.SettlesOnRegistration() {
		t.Error("only contado settles on registration")
	}
	if !domain.IsSaleRole(domain.RolInmobiliaria) || domain.IsSaleRole(domain.RolEscribano) {
		t.Error("sale roles should match reservation roles")
	}
}

func TestSaleListFilterNormalize(t *testing.T) {
	normalized, err := domain.SaleListFilter{DevelopmentID: " loteo ", Search: " ana "}.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if normalized.Page != domain.DefaultSalePage || normalized.Limit != domain.DefaultSaleLimit {
		t.Fatalf("Normalize() defaults = %d/%d", normalized.Page, normalized.Limit)
	}
	if normalized.DevelopmentID != "loteo" || normalized.Search != "ana" {
		t.Fatalf("Normalize() should trim, got %#v", normalized)
	}

	for name, filter := range map[string]domain.SaleListFilter{
		"page zero":       {Page: -1},
		"limit too large": {Limit: domain.MaxSaleLimit + 1},
	} {
		if _, err := filter.Normalize(); !errors.Is(err, domain.ErrSaleInvalidPage) {
			t.Errorf("%s: Normalize() error = %v, want %v", name, err, domain.ErrSaleInvalidPage)
		}
	}
	if _, err := (domain.SaleListFilter{States: []domain.SaleState{"vencida"}}).Normalize(); !errors.Is(err, domain.ErrSaleInvalidState) {
		t.Errorf("Normalize() error = %v, want %v", err, domain.ErrSaleInvalidState)
	}
}
