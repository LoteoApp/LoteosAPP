package dto

import "loteosapp/backend/internal/business/domain"

// UpdateSurveyorRequest is a partial update: a field omitted from the JSON
// body (nil after decoding) leaves the stored value unchanged.
type UpdateSurveyorRequest struct {
	Nombre   *string `json:"nombre,omitempty"`
	Apellido *string `json:"apellido,omitempty"`
}

type SurveyorResponse struct {
	domain.Usuario
}
