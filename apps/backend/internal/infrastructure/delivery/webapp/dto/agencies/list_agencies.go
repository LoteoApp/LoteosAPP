package dto

import "loteosapp/backend/internal/business/domain"

type ListAgenciesResponse struct {
	Agencies []domain.Agency `json:"inmobiliarias"`
}
