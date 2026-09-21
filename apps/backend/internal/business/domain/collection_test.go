package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
)

var collectionNow = time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)

func installmentsFixture() []domain.Installment {
	paidAt := collectionNow.AddDate(0, -1, 0)
	return []domain.Installment{
		{ID: "c-1", Numero: 1, Monto: 1000, Estado: domain.InstallmentStatePaid, FechaVencimiento: collectionNow.AddDate(0, -2, 0), FechaPago: &paidAt},
		{ID: "c-2", Numero: 2, Monto: 1000, Estado: domain.InstallmentStateOverdue, FechaVencimiento: collectionNow.AddDate(0, -1, 0)},
		{ID: "c-3", Numero: 3, Monto: 1000, Estado: domain.InstallmentStatePending, FechaVencimiento: collectionNow.AddDate(0, 1, 0)},
		{ID: "c-4", Numero: 4, Monto: 1000, Estado: domain.InstallmentStatePending, FechaVencimiento: collectionNow.AddDate(0, 2, 0)},
	}
}

func TestPaymentMediumAndInstallmentStateValidity(t *testing.T) {
	for _, medium := range []domain.PaymentMedium{domain.PaymentMediumCash, domain.PaymentMediumTransfer, domain.PaymentMediumCheque, domain.PaymentMediumOther} {
		if !medium.IsValid() {
			t.Errorf("%q should be valid", medium)
		}
	}
	if domain.PaymentMedium("bitcoin").IsValid() || domain.PaymentMedium("").IsValid() {
		t.Error("unknown medium should be invalid")
	}
	for _, state := range []domain.InstallmentState{domain.InstallmentStatePending, domain.InstallmentStatePaid, domain.InstallmentStateOverdue} {
		if !state.IsValid() {
			t.Errorf("%q should be valid", state)
		}
	}
	if domain.InstallmentState("cobrada").IsValid() {
		t.Error("unknown state should be invalid")
	}
	if !domain.IsCollectionRole(domain.RolInmobiliaria) || domain.IsCollectionRole(domain.RolAgrimensor) {
		t.Error("collection roles should match sale roles")
	}
}

func TestValidatePaymentTerms(t *testing.T) {
	cases := []struct {
		name  string
		terms domain.PaymentTerms
		want  error
	}{
		{"valid", domain.PaymentTerms{Medium: domain.PaymentMediumCash, PaidAt: collectionNow}, nil},
		{"skew tolerated", domain.PaymentTerms{Medium: domain.PaymentMediumCash, PaidAt: collectionNow.Add(time.Minute)}, nil},
		{"invalid medium", domain.PaymentTerms{Medium: "canje", PaidAt: collectionNow}, domain.ErrPaymentInvalidMedium},
		{"zero date", domain.PaymentTerms{Medium: domain.PaymentMediumCash}, domain.ErrPaymentInvalidDate},
		{"future date", domain.PaymentTerms{Medium: domain.PaymentMediumCash, PaidAt: collectionNow.Add(time.Hour)}, domain.ErrPaymentDateInFuture},
		{"observation too long", domain.PaymentTerms{Medium: domain.PaymentMediumCash, PaidAt: collectionNow, Observation: strings.Repeat("a", 501)}, domain.ErrPaymentObservationTooLong},
		{"observation at limit", domain.PaymentTerms{Medium: domain.PaymentMediumCash, PaidAt: collectionNow, Observation: strings.Repeat("á", 500)}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := domain.ValidatePaymentTerms(tc.terms, collectionNow); !errors.Is(err, tc.want) {
				t.Errorf("ValidatePaymentTerms() = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestEffectiveInstallmentState(t *testing.T) {
	paid := domain.Installment{Estado: domain.InstallmentStatePaid, FechaVencimiento: collectionNow.AddDate(0, -1, 0)}
	if got := domain.EffectiveInstallmentState(paid, collectionNow); got != domain.InstallmentStatePaid {
		t.Errorf("paid = %q", got)
	}
	overdue := domain.Installment{Estado: domain.InstallmentStatePending, FechaVencimiento: collectionNow.Add(-time.Second)}
	if got := domain.EffectiveInstallmentState(overdue, collectionNow); got != domain.InstallmentStateOverdue {
		t.Errorf("overdue = %q", got)
	}
	pending := domain.Installment{Estado: domain.InstallmentStatePending, FechaVencimiento: collectionNow.Add(time.Second)}
	if got := domain.EffectiveInstallmentState(pending, collectionNow); got != domain.InstallmentStatePending {
		t.Errorf("pending = %q", got)
	}
}

func TestComputeDebtSummary(t *testing.T) {
	entrega := &domain.DownPayment{Monto: 500.5, Estado: domain.InstallmentStatePending}
	summary := domain.ComputeDebtSummary(entrega, installmentsFixture())
	if summary.MontoTotal != 4500.5 || summary.MontoPagado != 1000 || summary.MontoPendiente != 3500.5 || summary.MontoVencido != 1000 {
		t.Errorf("amounts = %#v", summary)
	}
	if summary.CuotasPagadas != 1 || summary.CuotasPendientes != 3 || summary.CuotasVencidas != 1 {
		t.Errorf("counts = %#v", summary)
	}
	if summary.ProximoVencimiento == nil || !summary.ProximoVencimiento.Equal(collectionNow.AddDate(0, 1, 0)) {
		t.Errorf("next due = %v", summary.ProximoVencimiento)
	}

	paidEntrega := &domain.DownPayment{Monto: 500, Estado: domain.InstallmentStatePaid}
	allPaid := []domain.Installment{{Monto: 10, Estado: domain.InstallmentStatePaid}}
	settled := domain.ComputeDebtSummary(paidEntrega, allPaid)
	if settled.MontoPagado != 510 || settled.MontoPendiente != 0 || settled.ProximoVencimiento != nil {
		t.Errorf("settled = %#v", settled)
	}
	none := domain.ComputeDebtSummary(nil, nil)
	if none.MontoTotal != 0 || none.CuotasPendientes != 0 {
		t.Errorf("empty = %#v", none)
	}
}

func TestDebtStatementSaldada(t *testing.T) {
	statement := domain.DebtStatement{
		Entrega: &domain.DownPayment{Estado: domain.InstallmentStatePaid},
		Cuotas:  []domain.Installment{{Estado: domain.InstallmentStatePaid}},
	}
	if !statement.Saldada() {
		t.Error("everything paid should be saldada")
	}
	statement.Entrega.Estado = domain.InstallmentStatePending
	if statement.Saldada() {
		t.Error("pending entrega should not be saldada")
	}
	statement.Entrega = nil
	statement.Cuotas = append(statement.Cuotas, domain.Installment{Estado: domain.InstallmentStateOverdue})
	if statement.Saldada() {
		t.Error("overdue cuota should not be saldada")
	}
}

func TestSelectPaymentTakesTheFirstPendingInstallmentsInOrder(t *testing.T) {
	entrega := &domain.DownPayment{Monto: 250, Estado: domain.InstallmentStatePending}
	selection, err := domain.SelectPayment(entrega, installmentsFixture(), []string{" c-3 ", "c-2", ""}, true)
	if err != nil {
		t.Fatalf("SelectPayment() error = %v", err)
	}
	if !selection.IncludeDownPayment || selection.DownPaymentAmount != 250 || len(selection.Installments) != 2 {
		t.Fatalf("selection = %#v", selection)
	}
	if selection.Installments[0].ID != "c-2" || selection.Installments[1].ID != "c-3" {
		t.Errorf("installments = %#v, want c-2 then c-3", selection.Installments)
	}
	if selection.Amount() != 2250 {
		t.Errorf("amount = %v, want 2250", selection.Amount())
	}

	onlyEntrega, err := domain.SelectPayment(entrega, installmentsFixture(), nil, true)
	if err != nil || len(onlyEntrega.Installments) != 0 || onlyEntrega.Amount() != 250 {
		t.Errorf("entrega only = %#v, %v", onlyEntrega, err)
	}
}

func TestSelectPaymentRejections(t *testing.T) {
	installments := installmentsFixture()
	pendingEntrega := &domain.DownPayment{Monto: 250, Estado: domain.InstallmentStatePending}
	paidEntrega := &domain.DownPayment{Monto: 250, Estado: domain.InstallmentStatePaid}
	cases := []struct {
		name    string
		entrega *domain.DownPayment
		ids     []string
		include bool
		want    error
	}{
		{"nothing selected", pendingEntrega, nil, false, domain.ErrPaymentNothingSelected},
		{"blank ids only", pendingEntrega, []string{" "}, false, domain.ErrPaymentNothingSelected},
		{"unknown cuota", pendingEntrega, []string{"c-9"}, false, domain.ErrPaymentInstallmentUnknown},
		{"already paid", pendingEntrega, []string{"c-1"}, false, domain.ErrPaymentInstallmentPaid},
		{"skips a pending one", pendingEntrega, []string{"c-3"}, false, domain.ErrPaymentInstallmentOrder},
		{"skips in the middle", pendingEntrega, []string{"c-2", "c-4"}, false, domain.ErrPaymentInstallmentOrder},
		{"no entrega", nil, []string{"c-2"}, true, domain.ErrPaymentNoDownPayment},
		{"entrega already paid", paidEntrega, []string{"c-2"}, true, domain.ErrPaymentDownPaymentPaid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := domain.SelectPayment(tc.entrega, installments, tc.ids, tc.include); !errors.Is(err, tc.want) {
				t.Errorf("SelectPayment() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestSelectSettlement(t *testing.T) {
	entrega := &domain.DownPayment{Monto: 250, Estado: domain.InstallmentStatePending}
	expected := 3250.0
	selection, err := domain.SelectSettlement(entrega, installmentsFixture(), &expected)
	if err != nil {
		t.Fatalf("SelectSettlement() error = %v", err)
	}
	if !selection.IncludeDownPayment || len(selection.Installments) != 3 || selection.Amount() != 3250 {
		t.Errorf("selection = %#v", selection)
	}

	withoutCheck, err := domain.SelectSettlement(nil, installmentsFixture(), nil)
	if err != nil || withoutCheck.IncludeDownPayment || withoutCheck.Amount() != 3000 {
		t.Errorf("without entrega = %#v, %v", withoutCheck, err)
	}

	stale := 3000.0
	if _, err := domain.SelectSettlement(entrega, installmentsFixture(), &stale); !errors.Is(err, domain.ErrSettlementAmountMismatch) {
		t.Errorf("stale amount error = %v, want %v", err, domain.ErrSettlementAmountMismatch)
	}
	nearlyEqual := 3250.004
	if _, err := domain.SelectSettlement(entrega, installmentsFixture(), &nearlyEqual); err != nil {
		t.Errorf("rounding noise error = %v, want none", err)
	}

	paid := &domain.DownPayment{Monto: 250, Estado: domain.InstallmentStatePaid}
	if _, err := domain.SelectSettlement(paid, []domain.Installment{{Estado: domain.InstallmentStatePaid}}, nil); !errors.Is(err, domain.ErrSettlementNothingOwed) {
		t.Errorf("nothing owed error = %v, want %v", err, domain.ErrSettlementNothingOwed)
	}
}

func TestDueInstallmentFilterNormalize(t *testing.T) {
	filter, err := domain.DueInstallmentFilter{DevelopmentID: " loteo ", Search: " ana "}.Normalize()
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if filter.Page != domain.DefaultDueInstallmentPage || filter.Limit != domain.DefaultDueInstallmentLimit || filter.DevelopmentID != "loteo" || filter.Search != "ana" {
		t.Errorf("normalized = %#v", filter)
	}
	if len(filter.States) != 2 || filter.States[0] != domain.InstallmentStatePending || filter.States[1] != domain.InstallmentStateOverdue {
		t.Errorf("default states = %#v, want pendiente and vencida", filter.States)
	}

	explicit, err := domain.DueInstallmentFilter{States: []domain.InstallmentState{domain.InstallmentStatePaid}, Page: 2, Limit: 10}.Normalize()
	if err != nil || len(explicit.States) != 1 || explicit.Page != 2 || explicit.Limit != 10 {
		t.Errorf("explicit = %#v, %v", explicit, err)
	}

	from, to := collectionNow, collectionNow.AddDate(0, 1, 0)
	if _, err := (domain.DueInstallmentFilter{From: &from, To: &to}).Normalize(); err != nil {
		t.Errorf("valid period error = %v", err)
	}
	if _, err := (domain.DueInstallmentFilter{From: &to, To: &from}).Normalize(); !errors.Is(err, domain.ErrDueInstallmentInvalidPeriod) {
		t.Errorf("inverted period error = %v", err)
	}
	if _, err := (domain.DueInstallmentFilter{Page: -1}).Normalize(); !errors.Is(err, domain.ErrDueInstallmentInvalidPage) {
		t.Errorf("negative page error = %v", err)
	}
	if _, err := (domain.DueInstallmentFilter{Limit: domain.MaxDueInstallmentLimit + 1}).Normalize(); !errors.Is(err, domain.ErrDueInstallmentInvalidPage) {
		t.Errorf("limit too big error = %v", err)
	}
	if _, err := (domain.DueInstallmentFilter{States: []domain.InstallmentState{"cobrada"}}).Normalize(); !errors.Is(err, domain.ErrDueInstallmentInvalidState) {
		t.Errorf("invalid state error = %v", err)
	}
}
