package dto

import "loteosapp/backend/internal/business/domain"

type ListLoteosResponse struct {
	Loteos []domain.LoteoSummary `json:"loteos"`
}
