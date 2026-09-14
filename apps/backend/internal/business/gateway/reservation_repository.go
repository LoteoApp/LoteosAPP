package gateway

import (
	"context"
	"time"

	"loteosapp/backend/internal/business/domain"
)

type ReservationScope struct {
	AssigneeAuthProviderID *string
	ByAgencyAssignment     bool
	// ActorAuthProviderID is used only to calculate action permissions in a
	// response. It never narrows administrative reads.
	ActorAuthProviderID *string
}

type CreateReservationCommand struct {
	LoteoID                string
	LoteID                 string
	ClienteID              string
	VendedorID             string
	ActorID                string
	ActorAuthProviderID    string
	SellerIsActor          bool
	IdempotencyKey         string
	IdempotencyPayloadHash string
	CreatedAt              time.Time
}

type CancelReservationCommand struct {
	ReservationID       string
	ActorID             string
	ActorAuthProviderID string
	Reason              string
	CancelledAt         time.Time
}

type ReservationRepository interface {
	Create(ctx context.Context, command CreateReservationCommand) (domain.Reservation, error)
	List(ctx context.Context, filter domain.ReservationListFilter, scope ReservationScope) (domain.ReservationPage, error)
	Get(ctx context.Context, id string, scope ReservationScope) (domain.Reservation, error)
	Cancel(ctx context.Context, command CancelReservationCommand, scope ReservationScope) (domain.Reservation, error)
	ListEligibleSellers(ctx context.Context, loteoID string, scope ReservationScope) ([]domain.SellerOption, error)
	ExpireDue(ctx context.Context, now time.Time, limit int) (domain.ExpirationReport, error)
}
