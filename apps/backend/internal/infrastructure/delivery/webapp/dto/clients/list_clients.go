package dto

import "loteosapp/backend/internal/business/domain"

type ListClientsResponse struct {
	Clientes []domain.Cliente `json:"clientes"`
}
