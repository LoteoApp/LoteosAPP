package collections

import (
	"context"
	"strings"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ChargeInput is a cargo adicional as the request describes it. Currency is
// optional: blank means the sale's.
type ChargeInput struct {
	Type     string
	Amount   float64
	Currency string
	Detail   string
}

type RegisterPaymentInput struct {
	Actor              Actor
	SaleID             string
	InstallmentIDs     []string
	IncludeDownPayment bool
	Charges            []ChargeInput
	Medium             string
	// PaidAt is when the client paid; nil means now.
	PaidAt      *time.Time
	Observation string
}

// RegisterPayment collects the entrega and/or the next pending cuotas of a
// financed venta in one cobro. When nothing is left owed afterwards the
// venta is completada and its lote finalizado in the same transaction.
type RegisterPayment interface {
	Execute(ctx context.Context, input RegisterPaymentInput) (domain.Payment, error)
}

type registerPaymentUseCase struct {
	repository gateway.CollectionRepository
	users      gateway.UserRepository
	clock      Clock
}

func NewRegisterPayment(repository gateway.CollectionRepository, users gateway.UserRepository, clocks ...Clock) RegisterPayment {
	return &registerPaymentUseCase{repository: repository, users: users, clock: clockOrSystem(clocks)}
}

func (useCase *registerPaymentUseCase) Execute(ctx context.Context, input RegisterPaymentInput) (domain.Payment, error) {
	if !hasCollectionRole(input.Actor.Roles) {
		return domain.Payment{}, domain.ErrNoAutorizado
	}
	scope, err := collectionScope(input.Actor)
	if err != nil {
		return domain.Payment{}, err
	}
	saleID := strings.TrimSpace(input.SaleID)
	if saleID == "" {
		return domain.Payment{}, domain.ErrSaleNotFound
	}
	installmentIDs := nonBlank(input.InstallmentIDs)
	if !input.IncludeDownPayment && len(installmentIDs) == 0 {
		return domain.Payment{}, domain.ErrPaymentNothingSelected
	}
	now := useCase.clock.Now().UTC()
	terms, err := paymentTerms(input.Medium, input.PaidAt, input.Observation, now)
	if err != nil {
		return domain.Payment{}, err
	}
	charges, err := paymentCharges(input.Charges)
	if err != nil {
		return domain.Payment{}, err
	}
	actor, err := resolveActor(ctx, useCase.users, input.Actor)
	if err != nil {
		return domain.Payment{}, fromRepository(err)
	}
	if !hasCollectionRole([]string{string(actor.Rol)}) {
		return domain.Payment{}, domain.ErrNoAutorizado
	}
	payment, err := useCase.repository.RegisterPayment(ctx, gateway.RegisterPaymentCommand{
		SaleID:             saleID,
		ActorID:            actor.ID,
		Type:               domain.PaymentTypeRegular,
		InstallmentIDs:     installmentIDs,
		IncludeDownPayment: input.IncludeDownPayment,
		Charges:            charges,
		Medium:             terms.Medium,
		Observation:        terms.Observation,
		PaidAt:             terms.PaidAt,
		Now:                now,
	}, scope)
	if err != nil {
		return domain.Payment{}, fromRepository(err)
	}
	return payment, nil
}

func paymentTerms(medium string, paidAt *time.Time, observation string, now time.Time) (domain.PaymentTerms, error) {
	terms := domain.PaymentTerms{
		Medium:      domain.PaymentMedium(strings.TrimSpace(medium)),
		PaidAt:      now,
		Observation: strings.TrimSpace(observation),
	}
	if paidAt != nil {
		terms.PaidAt = paidAt.UTC()
	}
	if err := domain.ValidatePaymentTerms(terms, now); err != nil {
		return domain.PaymentTerms{}, err
	}
	return terms, nil
}

// paymentCharges maps and validates the cargos adicionales. The currency is
// left as typed (blank included): the repository resolves a blank one to the
// sale's currency, which only it knows.
func paymentCharges(charges []ChargeInput) ([]domain.PaymentChargeInput, error) {
	mapped := make([]domain.PaymentChargeInput, len(charges))
	for i, charge := range charges {
		mapped[i] = domain.PaymentChargeInput{
			Tipo:    domain.ChargeType(strings.TrimSpace(charge.Type)),
			Monto:   charge.Amount,
			Moneda:  charge.Currency,
			Detalle: charge.Detail,
		}
	}
	return domain.NormalizePaymentCharges(mapped, "")
}

func nonBlank(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
