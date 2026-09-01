package dto

import "loteosapp/backend/internal/business/domain"

type ListAgenciesResponse struct {
	Inmobiliarias []domain.Inmobiliaria `json:"inmobiliarias"`
}
