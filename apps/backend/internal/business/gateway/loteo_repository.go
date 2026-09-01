package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

// LoteoScope limits a loteo read to what one caller may see. A nil
// AssigneeAuthProviderID means no limit (administrador, administrativo).
// Otherwise the caller sees a loteo only through the assignment paths that
// are enabled: ByUserAssignment for a direct usuario_loteos row
// (agrimensor, escribano) and ByAgencyAssignment for one reached through the
// caller's inmobiliaria (inmobiliaria_loteos). The two are independent so a
// caller who holds several roles gets the union of what each grants, and an
// agrimensor tied to an agency by mistake still can't borrow its loteos.
type LoteoScope struct {
	AssigneeAuthProviderID *string
	ByUserAssignment       bool
	ByAgencyAssignment     bool
}

type LoteoRepository interface {
	// Create persists the loteo and its whole plan atomically: either every
	// polygon, manzana, lote and calle lands, or none does. The returned
	// Loteo lists manzanas, their lotes and calles in the same order as
	// loteo.Plan, so a caller can match each one back to the polygon it
	// sent.
	Create(ctx context.Context, actorAuthProviderID string, loteo domain.NewLoteo) (domain.Loteo, error)

	// List returns the active loteos as summaries, without geometry, ordered
	// by name. search, when non-empty, filters by nombre or ubicacion,
	// case-insensitively. scope limits the result to the loteos the caller
	// may see (see LoteoScope).
	List(ctx context.Context, search string, scope LoteoScope) ([]domain.LoteoSummary, error)

	// Get returns one loteo with its manzanas, lotes, calles and the geometry
	// of each (a nil polygon where an entity has no DXF ring yet). scope
	// limits visibility the same way as List: a loteo that doesn't exist,
	// isn't a valid id, or is outside the caller's scope all return
	// domain.ErrLoteoNotFound, so a caller can't tell which ids exist.
	Get(ctx context.Context, loteoID string, scope LoteoScope) (domain.Loteo, error)

	// UpdateLote sets the manually loaded values of one lot. loteoID scopes
	// the lookup so a caller authorized on one loteo can't reach a lot of
	// another by guessing its id; a lot that doesn't belong to loteoID
	// returns domain.ErrLoteNotFound.
	UpdateLote(ctx context.Context, actorAuthProviderID, loteoID, loteID string, data domain.LoteData) (domain.Lote, error)

	// IsAssignedToLoteo reports whether the user has the loteo assigned. It
	// answers only that question: what an assignment allows is a decision
	// for the use case.
	IsAssignedToLoteo(ctx context.Context, authProviderID, loteoID string) (bool, error)

	// LoteoExists reports whether a loteo with that id exists and is not soft
	// deleted. It lets a caller fail before doing work (e.g. an upload) for a
	// loteo that can't receive it.
	LoteoExists(ctx context.Context, loteoID string) (bool, error)

	// RecordDxfFile records the original DXF of a loteo in the archivos table,
	// superseding any DXF already recorded for it. The bytes must already be
	// in object storage under file.StorageKey. It returns
	// domain.ErrLoteoNotFound when loteoID names no loteo.
	RecordDxfFile(ctx context.Context, actorAuthProviderID, loteoID string, file domain.NewLoteoDxfFile) (domain.LoteoDxfFile, error)
}
