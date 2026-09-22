package gateway

import (
	"context"
	"time"

	"loteosapp/backend/internal/business/domain"
)

// RegisterPaymentCommand describes a cobro over a venta. A regular payment
// names the cuotas it pays and whether the entrega is included; a settlement
// pays everything still owed and carries the amount the caller saw, so the
// repository rejects it when the balance changed underneath.
type RegisterPaymentCommand struct {
	SaleID             string
	ActorID            string
	Type               domain.PaymentType
	InstallmentIDs     []string
	IncludeDownPayment bool
	ExpectedAmount     *float64
	// Charges are the cargos adicionales collected with this cobro (taxes,
	// administrative fees, services). A charge with no Moneda takes the
	// sale's; one in another currency is kept apart, never converted.
	Charges     []domain.PaymentChargeInput
	Medium      domain.PaymentMedium
	Observation string
	PaidAt      time.Time
	// Now is when the cobro is registered: it decides which cuotas are
	// vencidas in the returned data and stamps the estado rows.
	Now time.Time
}

// CollectionRepository reads and writes cobranza over ventas. Every method
// is bounded by the same SaleScope the ventas use, so an agency user only
// collects on the ventas of their own agency.
type CollectionRepository interface {
	GetDebtStatement(ctx context.Context, saleID string, scope SaleScope, now time.Time) (domain.DebtStatement, error)
	ListDueInstallments(ctx context.Context, filter domain.DueInstallmentFilter, scope SaleScope, now time.Time) (domain.DueInstallmentPage, error)
	RegisterPayment(ctx context.Context, command RegisterPaymentCommand, scope SaleScope) (domain.Payment, error)
}
