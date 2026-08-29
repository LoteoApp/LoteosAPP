package dto

import "loteosapp/backend/internal/business/domain"

// EntityRequest is one polygon read from a DXF layer by the frontend parser.
// Vertices lists each vertex once; repeating the first one at the end is
// accepted too.
type EntityRequest struct {
	Handle   string         `json:"handle"`
	Vertices domain.Polygon `json:"vertices"`
}

// ManzanaRequest carries the reference its lotes use to point at it. The
// reference only has to be unique within the request.
type ManzanaRequest struct {
	EntityRequest
	Ref string `json:"ref"`
}

// LoteRequest names the manzana the lot sits in, which the DXF itself doesn't
// record.
type LoteRequest struct {
	EntityRequest
	ManzanaRef string `json:"manzanaRef"`
}

type PlanRequest struct {
	Loteo    EntityRequest    `json:"loteo"`
	Manzanas []ManzanaRequest `json:"manzanas"`
	Lotes    []LoteRequest    `json:"lotes"`
	Calles   []EntityRequest  `json:"calles"`
}

// CreateLoteoRequest is the alta form plus, optionally, the geometry of its
// DXF. Plan is omitted when the loteo is registered before its plan exists.
type CreateLoteoRequest struct {
	Name        string       `json:"nombre"`
	Location    string       `json:"ubicacion"`
	Description string       `json:"descripcion"`
	Plan        *PlanRequest `json:"plano"`
}
