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
	LoteoID                string
	LoteID                 string
	ClienteID              string
	VendedorID             string
	ActorID                string
	PaymentMethod          domain.PaymentMethod
	IdempotencyKey         string
	IdempotencyPayloadHash string
	CreatedAt              time.Time
}

type SaleRepository interface {
	Create(ctx context.Context, command CreateSaleCommand) (domain.Sale, error)
	List(ctx context.Context, filter domain.SaleListFilter, scope SaleScope) (domain.SalePage, error)
	Get(ctx context.Context, id string, scope SaleScope) (domain.Sale, error)
}
