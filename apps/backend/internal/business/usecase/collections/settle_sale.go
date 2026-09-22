package collections

import (
	"context"
	"strings"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type SettleSaleInput struct {
	Actor  Actor
	SaleID string
	// ExpectedAmount is the saldo the caller saw; nil skips the check.
	ExpectedAmount *float64
	Charges        []ChargeInput
	Medium         string
	PaidAt         *time.Time
	Observation    string
}

// SettleSale collects everything still owed on a financed venta in one
// cobro (cancelación anticipada): the entrega if pending and every cuota not
// yet paid. The venta becomes completada and its lote finalizado.
type SettleSale interface {
	Execute(ctx context.Context, input SettleSaleInput) (domain.Payment, error)
}

type settleSaleUseCase struct {
	repository gateway.CollectionRepository
	users      gateway.UserRepository
	clock      Clock
}

func NewSettleSale(repository gateway.CollectionRepository, users gateway.UserRepository, clocks ...Clock) SettleSale {
	return &settleSaleUseCase{repository: repository, users: users, clock: clockOrSystem(clocks)}
}

func (useCase *settleSaleUseCase) Execute(ctx context.Context, input SettleSaleInput) (domain.Payment, error) {
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
		SaleID:         saleID,
		ActorID:        actor.ID,
		Type:           domain.PaymentTypeSettlement,
		ExpectedAmount: input.ExpectedAmount,
		Charges:        charges,
		Medium:         terms.Medium,
		Observation:    terms.Observation,
		PaidAt:         terms.PaidAt,
		Now:            now,
	}, scope)
	if err != nil {
		return domain.Payment{}, fromRepository(err)
	}
	return payment, nil
}
