package gateway

import (
	"context"
	"time"

	"loteosapp/backend/internal/business/domain"
)

// SaleScope narrows what an actor can read. An internal user reads every
// venta; an agency user only the ones sold by their own agency, which is read
// from the seller's usuarios.inmobiliaria_id since ventas stores no agency.
type SaleScope struct {
	AssigneeAuthProviderID *string
	ByAgency               bool
}

type CreateSaleCommand struct {
	DevelopmentID          string
	LotID                  string
	ClientID               string
	SellerID               string
	ActorID                string
	PaymentMethod          domain.PaymentMethod
	IdempotencyKey         string
	IdempotencyPayloadHash string
	// PaymentPlan is nil for contado; the repository builds and persists its
	// schedule over the lote price.
	PaymentPlan *domain.PaymentPlanInput
	CreatedAt   time.Time
}

// ConvertReservationCommand turns a reserva into a venta. Lote, cliente,
// vendedor, price and sale date are resolved by the repository from the
// locked reserva and lote, never taken from the caller; so is the actor's
// read scope, from their role as locked in the same transaction.
type ConvertReservationCommand struct {
	ReservationID          string
	ActorID                string
	ActorAuthProviderID    string
	PaymentMethod          domain.PaymentMethod
	PaymentPlan            *domain.PaymentPlanInput
	IdempotencyKey         string
	IdempotencyPayloadHash string
}

type SaleRepository interface {
	Create(ctx context.Context, command CreateSaleCommand) (domain.Sale, error)
	ConvertReservation(ctx context.Context, command ConvertReservationCommand) (domain.Sale, error)
	List(ctx context.Context, filter domain.SaleListFilter, scope SaleScope) (domain.SalePage, error)
	Get(ctx context.Context, id string, scope SaleScope) (domain.Sale, error)
}
