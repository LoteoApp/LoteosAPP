package sales

import (
	"context"
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

// PaymentPlanInput is the plan as the request describes it. TasaInteres is a
// percentage; MontoEntrega only applies to entrega_financiada.
type PaymentPlanInput struct {
	CantidadCuotas int
	TasaInteres    float64
	Periodicidad   string
	MontoEntrega   float64
}

type CreateSaleInput struct {
	Actor         Actor
	LoteoID       string
	LoteID        string
	ClienteID     string
	VendedorID    string
	PaymentMethod string
	// PaymentPlan is required for financiado and entrega_financiada and must
	// be absent for contado.
	PaymentPlan *PaymentPlanInput
}

// CreateSale registers a venta for an available lote and moves the lote to
// vendido. A financed sale also gets its plan de pago and cuotas, computed
// over the lote price in the same transaction. Unlike a reserva, an agency
// user may register the sale for a colleague of their agency, so the seller
// is never forced to the actor; the repository checks the seller belongs to
// the actor's agency.
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
	loteoID := strings.TrimSpace(input.LoteoID)
	loteID := strings.TrimSpace(input.LoteID)
	clienteID := strings.TrimSpace(input.ClienteID)
	sellerID := strings.TrimSpace(input.VendedorID)
	if loteoID == "" || loteID == "" {
		return domain.Sale{}, domain.ErrLoteNotFound
	}
	if clienteID == "" {
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
			CantidadCuotas: input.PaymentPlan.CantidadCuotas,
			TasaInteres:    input.PaymentPlan.TasaInteres,
			Periodicidad:   domain.PaymentPeriod(strings.TrimSpace(input.PaymentPlan.Periodicidad)),
			MontoEntrega:   input.PaymentPlan.MontoEntrega,
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
		LoteoID:       loteoID,
		LoteID:        loteID,
		ClienteID:     clienteID,
		VendedorID:    sellerID,
		ActorID:       actor.ID,
		PaymentMethod: method,
		PaymentPlan:   plan,
		CreatedAt:     useCase.clock.Now().UTC(),
	})
	if err != nil {
		return domain.Sale{}, fromRepository(err)
	}
	return sale, nil
}
