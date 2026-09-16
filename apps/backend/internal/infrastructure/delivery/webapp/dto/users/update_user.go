package dto

import "loteosapp/backend/internal/business/domain"

// UpdateUserRequest is a partial update: a field omitted from the JSON
// body (nil after decoding) leaves the stored value unchanged.
type UpdateUserRequest struct {
	Nombre   *string `json:"nombre,omitempty"`
	Apellido *string `json:"apellido,omitempty"`
}

type UserResponse struct {
	domain.Usuario
}
