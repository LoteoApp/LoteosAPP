package dto

type CreateSaleRequest struct {
	ClienteID     string                    `json:"clienteId"`
	VendedorID    string                    `json:"vendedorId"`
	ModalidadPago string                    `json:"modalidadPago,omitempty"`
	PlanPago      *CreateSalePaymentPlanDTO `json:"planPago,omitempty"`
}

// CreateSalePaymentPlanDTO describes the financing of a sale. TasaInteres is
// a percentage over the financed amount and MontoEntrega only applies to
// entrega_financiada. Due dates derive from the sale date, so none is sent.
type CreateSalePaymentPlanDTO struct {
	CantidadCuotas int     `json:"cantidadCuotas"`
	TasaInteres    float64 `json:"tasaInteres"`
	Periodicidad   string  `json:"periodicidad"`
	MontoEntrega   float64 `json:"montoEntrega"`
}
