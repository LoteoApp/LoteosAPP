package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

type LoteoRepository interface {
	// Create persists the loteo and its whole plan atomically: either every
	// polygon, manzana, lote and calle lands, or none does. The returned
	// Loteo lists manzanas, their lotes and calles in the same order as
	// loteo.Plan, so a caller can match each one back to the polygon it
	// sent.
	Create(ctx context.Context, actorAuthProviderID string, loteo domain.NewLoteo) (domain.Loteo, error)

	// UpdateLote sets the manually loaded values of one lot. loteoID scopes
	// the lookup so a caller authorized on one loteo can't reach a lot of
	// another by guessing its id; a lot that doesn't belong to loteoID
	// returns domain.ErrLoteNotFound.
	UpdateLote(ctx context.Context, actorAuthProviderID, loteoID, loteID string, data domain.LoteData) (domain.Lote, error)

	// IsAssignedToLoteo reports whether the user has the loteo assigned. It
	// answers only that question: what an assignment allows is a decision
	// for the use case.
	IsAssignedToLoteo(ctx context.Context, authProviderID, loteoID string) (bool, error)
}
