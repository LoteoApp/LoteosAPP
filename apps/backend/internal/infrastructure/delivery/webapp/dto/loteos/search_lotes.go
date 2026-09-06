package dto

import "loteosapp/backend/internal/business/domain"

type SearchLotesResponse struct {
	Lotes []domain.LoteSummary `json:"lotes"`
}
