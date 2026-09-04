package dto

import "loteosapp/backend/internal/business/domain"

type CreateAgencyRequest struct {
	BusinessName string  `json:"razonSocial"`
	CUIT         *string `json:"cuit,omitempty"`
	Phone        *string `json:"telefono,omitempty"`
	Email        *string `json:"email,omitempty"`
}

type AgencyResponse struct {
	domain.Agency
}
