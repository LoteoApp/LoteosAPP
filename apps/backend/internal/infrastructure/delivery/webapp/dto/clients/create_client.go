package dto

import "loteosapp/backend/internal/business/domain"

type CreateClientRequest struct {
	Nombre   string  `json:"nombre"`
	Apellido string  `json:"apellido"`
	DNI      string  `json:"dni"`
	Celular  *string `json:"celular,omitempty"`
	Email    *string `json:"email,omitempty"`
}

type ClientResponse struct {
	domain.Cliente
}
