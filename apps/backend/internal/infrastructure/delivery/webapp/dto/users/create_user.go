package dto

import "loteosapp/backend/internal/business/domain"

type CreateUserRequest struct {
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
	Email    string `json:"email"`
	Rol      string `json:"rol"`
	// InmobiliariaID is required when Rol is inmobiliaria and must be
	// omitted otherwise.
	InmobiliariaID string `json:"inmobiliariaId,omitempty"`
}

type CreateUserResponse struct {
	domain.Usuario
	TemporaryPassword string `json:"temporaryPassword"`
}
