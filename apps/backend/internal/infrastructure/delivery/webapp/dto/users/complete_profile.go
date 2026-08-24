package dto

import "loteosapp/backend/internal/business/domain"

type CompleteProfileRequest struct {
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
}

type CompleteProfileResponse struct {
	domain.Usuario
}
