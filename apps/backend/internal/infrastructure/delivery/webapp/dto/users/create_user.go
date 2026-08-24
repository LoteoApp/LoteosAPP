package dto

import "loteosapp/backend/internal/business/domain"

type CreateUserRequest struct {
	Email string `json:"email"`
	Rol   string `json:"rol"`
}

type CreateUserResponse struct {
	domain.Usuario
	TemporaryPassword string `json:"temporaryPassword"`
}
