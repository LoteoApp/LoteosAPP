package dto

import "loteosapp/backend/internal/business/domain"

type CreateSurveyorRequest struct {
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
	Email    string `json:"email"`
}

type CreateSurveyorResponse struct {
	domain.Usuario
	TemporaryPassword string `json:"temporaryPassword"`
}
