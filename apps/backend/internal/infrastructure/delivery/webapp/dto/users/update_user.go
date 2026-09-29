package dto

import "loteosapp/backend/internal/business/domain"

// UpdateUserRequest is a partial update: a field omitted from the JSON
// body (nil after decoding) leaves the stored value unchanged.
type UpdateUserRequest struct {
	Nombre   *string `json:"nombre,omitempty"`
	Apellido *string `json:"apellido,omitempty"`
	// InmobiliariaID only fills in a missing agency for a rol inmobiliaria
	// user; the backend rejects it for any other case.
	InmobiliariaID *string `json:"inmobiliariaId,omitempty"`
}

type UserResponse struct {
	domain.Usuario
}
