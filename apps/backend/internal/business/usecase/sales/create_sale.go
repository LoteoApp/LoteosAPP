package sales

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

// PaymentPlanInput is the plan as the request describes it. InterestRate is a
// percentage; DownPayment only applies to entrega_financiada.
type PaymentPlanInput struct {
	Installments int
	InterestRate float64
	Period       string
	DownPayment  float64
}

type CreateSaleInput struct {
	Actor          Actor
	DevelopmentID  string
	LotID          string
	ClientID       string
	SellerID       string
	PaymentMethod  string
	IdempotencyKey string
	// PaymentPlan is required for financiado and entrega_financiada and must
	// be absent for contado.
	PaymentPlan *PaymentPlanInput
}

// CreateSale registers a venta for an available lote and moves the lote to
// vendido. A financed sale also gets its plan de pago and cuotas, computed
// over the lote price in the same transaction. Unlike a reserva, an agency
// user may register the sale for a colleague of their agency, so the seller
// is never forced to the actor; the repository checks the seller belongs to
// the actor's agency and that the agency is assigned to the loteo. A retry
// with the same Idempotency-Key and payload returns the venta already
// registered.
type CreateSale interface {
	Execute(ctx context.Context, input CreateSaleInput) (domain.Sale, error)
}

type createSaleUseCase struct {
	repository gateway.SaleRepository
	users      gateway.UserRepository
	clock      Clock
}

func NewCreateSale(repository gateway.SaleRepository, users gateway.UserRepository, clocks ...Clock) CreateSale {
	clock := Clock(SystemClock{})
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &createSaleUseCase{repository: repository, users: users, clock: clock}
}

func (useCase *createSaleUseCase) Execute(ctx context.Context, input CreateSaleInput) (domain.Sale, error) {
	if !hasSaleRole(input.Actor.Roles) {
		return domain.Sale{}, domain.ErrNoAutorizado
	}
	key, err := domain.NormalizeIdempotencyKey(input.IdempotencyKey)
	if err != nil {
		return domain.Sale{}, domain.ErrSaleIdempotencyRequired
	}
	developmentID := strings.TrimSpace(input.DevelopmentID)
	lotID := strings.TrimSpace(input.LotID)
	clientID := strings.TrimSpace(input.ClientID)
	sellerID := strings.TrimSpace(input.SellerID)
	if developmentID == "" || lotID == "" {
		return domain.Sale{}, domain.ErrLoteNotFound
	}
	if clientID == "" {
		return domain.Sale{}, domain.ErrSaleInvalidClient
	}
	if sellerID == "" {
		return domain.Sale{}, domain.ErrSaleSellerRequired
	}
	method := domain.PaymentMethod(strings.TrimSpace(input.PaymentMethod))
	if method == "" {
		method = domain.PaymentMethodCash
	}
	if !method.IsValid() {
		return domain.Sale{}, domain.ErrSaleInvalidPaymentMethod
	}
	var plan *domain.PaymentPlanInput
	if input.PaymentPlan != nil {
		plan = &domain.PaymentPlanInput{
			Installments: input.PaymentPlan.Installments,
			InterestRate: input.PaymentPlan.InterestRate,
			Period:       domain.PaymentPeriod(strings.TrimSpace(input.PaymentPlan.Period)),
			DownPayment:  input.PaymentPlan.DownPayment,
		}
	}
	if err := domain.ValidatePaymentPlan(method, plan); err != nil {
		return domain.Sale{}, err
	}

	actor, err := resolveActor(ctx, useCase.users, input.Actor)
	if err != nil {
		return domain.Sale{}, fromRepository(err)
	}
	if !hasSaleRole([]string{string(actor.Rol)}) {
		return domain.Sale{}, domain.ErrNoAutorizado
	}

	sale, err := useCase.repository.Create(ctx, gateway.CreateSaleCommand{
		DevelopmentID:          developmentID,
		LotID:                  lotID,
		ClientID:               clientID,
		SellerID:               sellerID,
		ActorID:                actor.ID,
		PaymentMethod:          method,
		IdempotencyKey:         key,
		IdempotencyPayloadHash: salePayloadHash(developmentID, lotID, clientID, sellerID, method, plan),
		PaymentPlan:            plan,
		CreatedAt:              useCase.clock.Now().UTC(),
	})
	if err != nil {
		return domain.Sale{}, fromRepository(err)
	}
	return sale, nil
}

func salePayloadHash(developmentID, lotID, clientID, sellerID string, method domain.PaymentMethod, plan *domain.PaymentPlanInput) string {
	payload := fmt.Sprintf("%d:%s%d:%s%d:%s%d:%s%d:%s",
		len(developmentID), developmentID, len(lotID), lotID, len(clientID), clientID,
		len(sellerID), sellerID, len(method), method)
	if plan != nil {
		payload += fmt.Sprintf("|%d:%g:%s:%g", plan.Installments, plan.InterestRate, plan.Period, plan.DownPayment)
	}
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}
