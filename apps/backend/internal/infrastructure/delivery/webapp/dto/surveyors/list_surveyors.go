package dto

import "loteosapp/backend/internal/business/domain"

type ListSurveyorsResponse struct {
	Agrimensores []domain.Usuario `json:"agrimensores"`
}
