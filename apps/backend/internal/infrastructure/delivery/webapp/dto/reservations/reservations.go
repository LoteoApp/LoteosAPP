package dto

import "loteosapp/backend/internal/business/domain"

type CreateReservationRequest struct {
	ClienteID  string `json:"clienteId"`
	VendedorID string `json:"vendedorId,omitempty"`
}

type CancelReservationRequest struct {
	Razon string `json:"razon"`
}

type SellersResponse struct {
	Vendedores []domain.SellerOption `json:"vendedores"`
}

type ReservationResponse struct {
	domain.Reservation
}
