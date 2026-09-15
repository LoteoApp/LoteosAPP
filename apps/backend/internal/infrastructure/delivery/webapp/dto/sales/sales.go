package dto

type CreateSaleRequest struct {
	ClienteID     string `json:"clienteId"`
	VendedorID    string `json:"vendedorId"`
	ModalidadPago string `json:"modalidadPago,omitempty"`
}
