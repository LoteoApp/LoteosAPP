package handler

import (
	"net/http"

	"loteosapp/backend/internal/business/usecase/loteos"
	dto "loteosapp/backend/internal/infrastructure/delivery/webapp/dto/loteos"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/middleware"
	"loteosapp/backend/internal/infrastructure/delivery/webapp/response"
)

// A plan is by far the largest body the API accepts: thousands of polygons of
// a few vertices each. The cap bounds how much a single request can make the
// decoder allocate, which the per-polygon limits in the domain can't do
// because they only apply once the whole body is already parsed.
const maxCreateLoteoBytes = 16 << 20

type CreateLoteoHandler struct {
	createLoteo loteos.CreateLoteo
}

func NewCreateLoteoHandler(createLoteo loteos.CreateLoteo) *CreateLoteoHandler {
	return &CreateLoteoHandler{createLoteo: createLoteo}
}

// Handle registers a loteo with the geometry the frontend read from its DXF.
// It must run behind middleware.RequireAuth.
func (handler *CreateLoteoHandler) Handle(w http.ResponseWriter, request *http.Request) error {
	// PrincipalFromContext is always populated here: this handler only ever
	// runs behind middleware.RequireAuth.
	principal, _ := middleware.PrincipalFromContext(request.Context())

	request.Body = http.MaxBytesReader(w, request.Body, maxCreateLoteoBytes)

	body, err := decodeJSON[dto.CreateLoteoRequest](request)
	if err != nil {
		return err
	}

	actor := loteos.Actor{AuthProviderID: principal.Subject, Roles: principal.Roles}

	loteo, err := handler.createLoteo.Execute(request.Context(), actor, toLoteoInput(body))
	if err != nil {
		return err
	}

	response.WriteJSON(w, http.StatusCreated, loteo)
	return nil
}

func toLoteoInput(body dto.CreateLoteoRequest) loteos.LoteoInput {
	input := loteos.LoteoInput{
		Name:        body.Name,
		Location:    body.Location,
		Description: body.Description,
	}
	if body.Plan == nil {
		return input
	}

	plan := &loteos.PlanInput{
		Loteo:    toEntityInput(body.Plan.Loteo),
		Manzanas: make([]loteos.ManzanaInput, 0, len(body.Plan.Manzanas)),
		Lotes:    make([]loteos.LoteInput, 0, len(body.Plan.Lotes)),
		Calles:   make([]loteos.EntityInput, 0, len(body.Plan.Calles)),
	}

	for _, manzana := range body.Plan.Manzanas {
		plan.Manzanas = append(plan.Manzanas, loteos.ManzanaInput{
			EntityInput: toEntityInput(manzana.EntityRequest),
			Ref:         manzana.Ref,
		})
	}
	for _, lote := range body.Plan.Lotes {
		plan.Lotes = append(plan.Lotes, loteos.LoteInput{
			EntityInput: toEntityInput(lote.EntityRequest),
			ManzanaRef:  lote.ManzanaRef,
		})
	}
	for _, calle := range body.Plan.Calles {
		plan.Calles = append(plan.Calles, toEntityInput(calle))
	}

	input.Plan = plan

	return input
}

func toEntityInput(entity dto.EntityRequest) loteos.EntityInput {
	return loteos.EntityInput{Handle: entity.Handle, Polygon: entity.Vertices}
}
