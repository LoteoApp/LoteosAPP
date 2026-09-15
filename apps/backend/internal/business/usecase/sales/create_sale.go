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

type CreateSaleInput struct {
	Actor          Actor
	LoteoID        string
	LoteID         string
	ClienteID      string
	VendedorID     string
	PaymentMethod  string
	IdempotencyKey string
}

// CreateSale registers a venta al contado for an available lote and moves
// the lote to vendido. Unlike a reserva, an agency user may register the
// sale for a colleague of their agency, so the seller is never forced to
// the actor; the repository checks the seller belongs to the actor's agency
// and that the agency is assigned to the loteo. A retry with the same
// Idempotency-Key and payload returns the venta already registered.
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
		return domain.Sale{}, err
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
	if !method.IsAvailable() {
		return domain.Sale{}, domain.ErrSalePaymentMethodUnavailable
	}

	actor, err := resolveActor(ctx, useCase.users, input.Actor)
	if err != nil {
		return domain.Sale{}, fromRepository(err)
	}
	if !hasSaleRole([]string{string(actor.Rol)}) {
		return domain.Sale{}, domain.ErrNoAutorizado
	}

	sale, err := useCase.repository.Create(ctx, gateway.CreateSaleCommand{
		LoteoID:                loteoID,
		LoteID:                 loteID,
		ClienteID:              clienteID,
		VendedorID:             sellerID,
		ActorID:                actor.ID,
		PaymentMethod:          method,
		IdempotencyKey:         key,
		IdempotencyPayloadHash: salePayloadHash(loteoID, loteID, clienteID, sellerID, method),
		CreatedAt:              useCase.clock.Now().UTC(),
	})
	if err != nil {
		return domain.Sale{}, fromRepository(err)
	}
	return sale, nil
}

func salePayloadHash(loteoID, loteID, clienteID, sellerID string, method domain.PaymentMethod) string {
	payload := fmt.Sprintf("%d:%s%d:%s%d:%s%d:%s%d:%s",
		len(loteoID), loteoID, len(loteID), loteID, len(clienteID), clienteID,
		len(sellerID), sellerID, len(method), method)
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}
