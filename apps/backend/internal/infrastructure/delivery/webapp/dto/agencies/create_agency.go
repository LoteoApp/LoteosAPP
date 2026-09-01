package dto

import "loteosapp/backend/internal/business/domain"

type CreateAgencyRequest struct {
	RazonSocial string  `json:"razonSocial"`
	CUIT        *string `json:"cuit,omitempty"`
	Telefono    *string `json:"telefono,omitempty"`
	Email       *string `json:"email,omitempty"`
}

type AgencyResponse struct {
	domain.Inmobiliaria
}
