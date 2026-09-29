package domain_test

import (
	"errors"
	"strings"
	"testing"

	"loteosapp/backend/internal/business/domain"
)

func TestChargeTypeIsValid(t *testing.T) {
	for _, charge := range []domain.ChargeType{
		domain.ChargeTypeMunicipalTax, domain.ChargeTypeProvincialTax, domain.ChargeTypeAdministrative,
		domain.ChargeTypeFees, domain.ChargeTypeServices, domain.ChargeTypeAgency, domain.ChargeTypeOther,
	} {
		if !charge.IsValid() {
			t.Errorf("%q should be valid", charge)
		}
	}
	if domain.ChargeType("propina").IsValid() || domain.ChargeType("").IsValid() {
		t.Error("unknown charge type should be invalid")
	}
}

func TestNormalizeCurrency(t *testing.T) {
	if got := domain.NormalizeCurrency(" ars ", "USD"); got != "ARS" {
		t.Errorf("NormalizeCurrency() = %q, want ARS", got)
	}
	if got := domain.NormalizeCurrency("  ", " usd "); got != "USD" {
		t.Errorf("blank currency = %q, want the fallback USD", got)
	}
	if got := domain.NormalizeCurrency("", ""); got != "" {
		t.Errorf("no currency at all = %q, want empty", got)
	}
}

func TestNormalizePaymentCharges(t *testing.T) {
	charges, err := domain.NormalizePaymentCharges([]domain.PaymentChargeInput{
		{Type: domain.ChargeTypeServices, Amount: 150000.5, Currency: " ars ", Detail: "  Agua y luz  "},
		{Type: domain.ChargeTypeAdministrative, Amount: 25},
	}, "usd")
	if err != nil {
		t.Fatalf("NormalizePaymentCharges() error = %v", err)
	}
	if len(charges) != 2 {
		t.Fatalf("charges = %#v", charges)
	}
	if charges[0].Currency != "ARS" || charges[0].Amount != 150000.5 || charges[0].Detail != "Agua y luz" {
		t.Errorf("first charge = %#v", charges[0])
	}
	// A charge without its own currency takes the sale's.
	if charges[1].Currency != "USD" || charges[1].Type != domain.ChargeTypeAdministrative {
		t.Errorf("second charge = %#v", charges[1])
	}

	empty, err := domain.NormalizePaymentCharges(nil, "USD")
	if err != nil || len(empty) != 0 {
		t.Errorf("no charges = %#v, %v", empty, err)
	}
}

func TestNormalizePaymentChargesRejections(t *testing.T) {
	valid := domain.PaymentChargeInput{Type: domain.ChargeTypeServices, Amount: 100, Currency: "ARS"}
	cases := []struct {
		name   string
		charge domain.PaymentChargeInput
		want   error
	}{
		{"unknown type", domain.PaymentChargeInput{Type: "propina", Amount: 100}, domain.ErrChargeInvalidType},
		{"zero amount", domain.PaymentChargeInput{Type: domain.ChargeTypeServices, Amount: 0}, domain.ErrChargeInvalidAmount},
		{"negative amount", domain.PaymentChargeInput{Type: domain.ChargeTypeServices, Amount: -10}, domain.ErrChargeInvalidAmount},
		{"too many decimals", domain.PaymentChargeInput{Type: domain.ChargeTypeServices, Amount: 10.555}, domain.ErrChargeInvalidAmount},
		{"currency too long", domain.PaymentChargeInput{Type: domain.ChargeTypeServices, Amount: 10, Currency: "PESOS ARGENTINOS"}, domain.ErrChargeInvalidCurrency},
		{"detail too long", domain.PaymentChargeInput{Type: domain.ChargeTypeServices, Amount: 10, Detail: strings.Repeat("x", 201)}, domain.ErrChargeDetailTooLong},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := domain.NormalizePaymentCharges([]domain.PaymentChargeInput{tc.charge}, "USD"); !errors.Is(err, tc.want) {
				t.Errorf("NormalizePaymentCharges() error = %v, want %v", err, tc.want)
			}
		})
	}

	tooMany := make([]domain.PaymentChargeInput, domain.MaxPaymentCharges+1)
	for i := range tooMany {
		tooMany[i] = valid
	}
	if _, err := domain.NormalizePaymentCharges(tooMany, "USD"); !errors.Is(err, domain.ErrTooManyCharges) {
		t.Errorf("too many charges error = %v, want %v", err, domain.ErrTooManyCharges)
	}
	atLimit := tooMany[:domain.MaxPaymentCharges]
	if _, err := domain.NormalizePaymentCharges(atLimit, "USD"); err != nil {
		t.Errorf("charges at the limit error = %v, want none", err)
	}
}

func TestPaymentTotalsKeepsEachCurrencyApart(t *testing.T) {
	charges := []domain.PaymentCharge{
		{Type: domain.ChargeTypeServices, Amount: 150000, Currency: "ARS"},
		{Type: domain.ChargeTypeMunicipalTax, Amount: 12000.5, Currency: "ARS"},
		{Type: domain.ChargeTypeFees, Amount: 300, Currency: "USD"},
		{Type: domain.ChargeTypeOther, Amount: 40, Currency: "EUR"},
	}
	totals := domain.PaymentTotals(20000, "USD", charges)
	want := []domain.CurrencyTotal{
		{Currency: "USD", Amount: 20300},
		{Currency: "ARS", Amount: 162000.5},
		{Currency: "EUR", Amount: 40},
	}
	if len(totals) != len(want) {
		t.Fatalf("totals = %#v, want %#v", totals, want)
	}
	for i, total := range totals {
		if total != want[i] {
			t.Errorf("total %d = %#v, want %#v", i, total, want[i])
		}
	}

	// Without charges there is one total, the plan's.
	plain := domain.PaymentTotals(500, "usd", nil)
	if len(plain) != 1 || plain[0] != (domain.CurrencyTotal{Currency: "USD", Amount: 500}) {
		t.Errorf("plain totals = %#v", plain)
	}
	// A cobro of charges only (no amount applied to the plan) has no entry
	// for the sale's currency.
	onlyCharges := domain.PaymentTotals(0, "USD", []domain.PaymentCharge{{Amount: 10, Currency: "ARS"}})
	if len(onlyCharges) != 1 || onlyCharges[0].Currency != "ARS" {
		t.Errorf("charges-only totals = %#v", onlyCharges)
	}
	if got := domain.PaymentTotals(0, "USD", nil); len(got) != 0 {
		t.Errorf("empty totals = %#v", got)
	}
}

func TestChargeTotals(t *testing.T) {
	totals := domain.ChargeTotals([]domain.PaymentCharge{
		{Amount: 1000, Currency: "ARS"},
		{Amount: 500.25, Currency: "ARS"},
		{Amount: 20, Currency: "USD"},
	})
	want := []domain.CurrencyTotal{{Currency: "ARS", Amount: 1500.25}, {Currency: "USD", Amount: 20}}
	if len(totals) != 2 || totals[0] != want[0] || totals[1] != want[1] {
		t.Errorf("ChargeTotals() = %#v, want %#v", totals, want)
	}
	if got := domain.ChargeTotals(nil); len(got) != 0 {
		t.Errorf("ChargeTotals(nil) = %#v, want empty", got)
	}
}
