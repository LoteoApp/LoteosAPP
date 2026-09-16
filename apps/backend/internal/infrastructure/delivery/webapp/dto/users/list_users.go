package dto

import "loteosapp/backend/internal/business/domain"

type ListUsersResponse struct {
	Usuarios []domain.Usuario `json:"usuarios"`
}
