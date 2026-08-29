package loteos

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// Actor is the authenticated caller a use case authorizes against.
// AuthProviderID is the identity provider's user ID (the JWT subject).
type Actor struct {
	AuthProviderID string
	Roles          []string
}

// EntityInput is a polygon as it arrives from the client, before validation.
type EntityInput struct {
	Handle  string
	Polygon domain.Polygon
}

// ManzanaInput adds the reference the client uses to point its lotes at this
// manzana. The reference is meaningful only inside one request: it's
// resolved to a position here and never persisted.
type ManzanaInput struct {
	EntityInput
	Ref string
}

// LoteInput names the manzana the lot sits in. The DXF parser doesn't build
// that relationship, so the client sends it and the backend checks that it
// points at a manzana present in the same plan.
type LoteInput struct {
	EntityInput
	ManzanaRef string
}

type PlanInput struct {
	Loteo    EntityInput
	Manzanas []ManzanaInput
	Lotes    []LoteInput
	Calles   []EntityInput
}

// LoteoInput is the alta form plus, optionally, the geometry parsed from a
// DXF. Plan is nil when the loteo is registered before its DXF exists.
type LoteoInput struct {
	Name        string
	Location    string
	Description string
	Plan        *PlanInput
}

// CreateLoteo registers a loteo and, when the caller sends one, the geometry
// the frontend read from its DXF. Only an administrador may do this.
type CreateLoteo interface {
	Execute(ctx context.Context, actor Actor, input LoteoInput) (domain.Loteo, error)
}

type createLoteoUseCase struct {
	repository gateway.LoteoRepository
}

func NewCreateLoteo(repository gateway.LoteoRepository) CreateLoteo {
	return &createLoteoUseCase{repository: repository}
}

// Execute validates the plan before it reaches persistence: every ring has to
// be a usable polygon and every lote has to name a manzana of this same plan,
// so a hand-crafted request can't hang lots off a manzana of another loteo.
func (useCase *createLoteoUseCase) Execute(
	ctx context.Context,
	actor Actor,
	input LoteoInput,
) (domain.Loteo, error) {
	// An agrimensor works on the loteos assigned to them, and a loteo that
	// doesn't exist yet can't be assigned, so creating one is administrador
	// only.
	if !domain.HasRole(actor.Roles, domain.RolAdministrador) {
		return domain.Loteo{}, domain.ErrNoAutorizado
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Loteo{}, domain.ErrInvalidLoteoName
	}

	plan, err := normalizePlan(input.Plan)
	if err != nil {
		return domain.Loteo{}, err
	}

	loteo, err := useCase.repository.Create(ctx, actor.AuthProviderID, domain.NewLoteo{
		Name:        name,
		Location:    strings.TrimSpace(input.Location),
		Description: strings.TrimSpace(input.Description),
		Plan:        plan,
	})
	if err != nil {
		return domain.Loteo{}, fromRepository(err)
	}

	return loteo, nil
}

func normalizePlan(input *PlanInput) (*domain.DxfPlan, error) {
	if input == nil {
		return nil, nil
	}

	if err := checkPlanSize(input); err != nil {
		return nil, err
	}

	if len(input.Loteo.Polygon) == 0 {
		return nil, domain.ErrPlanWithoutLoteo
	}
	loteo, err := normalizeEntity(input.Loteo)
	if err != nil {
		return nil, err
	}

	manzanas := make([]domain.DxfEntity, 0, len(input.Manzanas))
	positionByRef := make(map[string]int, len(input.Manzanas))
	for _, manzana := range input.Manzanas {
		ref := strings.TrimSpace(manzana.Ref)
		if _, repeated := positionByRef[ref]; ref == "" || repeated {
			return nil, domain.ErrInvalidManzanaRef
		}

		entity, err := normalizeEntity(manzana.EntityInput)
		if err != nil {
			return nil, err
		}

		positionByRef[ref] = len(manzanas)
		manzanas = append(manzanas, entity)
	}

	lotes := make([]domain.DxfLote, 0, len(input.Lotes))
	for _, lote := range input.Lotes {
		position, exists := positionByRef[strings.TrimSpace(lote.ManzanaRef)]
		if !exists {
			return nil, domain.ErrUnknownManzana
		}

		entity, err := normalizeEntity(lote.EntityInput)
		if err != nil {
			return nil, err
		}

		lotes = append(lotes, domain.DxfLote{DxfEntity: entity, ManzanaIndex: position})
	}

	calles := make([]domain.DxfEntity, 0, len(input.Calles))
	for _, calle := range input.Calles {
		entity, err := normalizeEntity(calle)
		if err != nil {
			return nil, err
		}

		calles = append(calles, entity)
	}

	return &domain.DxfPlan{Loteo: loteo, Manzanas: manzanas, Lotes: lotes, Calles: calles}, nil
}

// checkPlanSize bounds both the polygons and the vertices of a plan before a
// single ring is validated: checking a ring is quadratic in its vertices, so
// the total has to be capped before that work starts.
func checkPlanSize(input *PlanInput) error {
	if 1+len(input.Manzanas)+len(input.Lotes)+len(input.Calles) > domain.MaxPolygonsPerPlan {
		return domain.ErrPlanTooLarge
	}

	vertices := len(input.Loteo.Polygon)
	for _, manzana := range input.Manzanas {
		vertices += len(manzana.Polygon)
	}
	for _, lote := range input.Lotes {
		vertices += len(lote.Polygon)
	}
	for _, calle := range input.Calles {
		vertices += len(calle.Polygon)
	}
	if vertices > domain.MaxVerticesPerPlan {
		return domain.ErrPlanTooLarge
	}

	return nil
}

func normalizeEntity(input EntityInput) (domain.DxfEntity, error) {
	polygon, err := input.Polygon.Normalize()
	if err != nil {
		return domain.DxfEntity{}, err
	}

	return domain.DxfEntity{Handle: strings.TrimSpace(input.Handle), Polygon: polygon}, nil
}
