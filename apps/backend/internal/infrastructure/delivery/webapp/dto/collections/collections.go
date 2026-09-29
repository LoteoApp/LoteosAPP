package dto

// ChargeDTO is a cargo adicional collected with a cobro (taxes,
// administrative fees, professional fees, services...). Currency is optional:
// blank means the sale's currency, and a different one is collected apart
// without being converted.
type ChargeDTO struct {
	Type     string  `json:"tipo"`
	Amount   float64 `json:"monto"`
	Currency string  `json:"moneda,omitempty"`
	Detail   string  `json:"detalle,omitempty"`
}

// RegisterPaymentRequest collects the entrega and/or the next pending
// cuotas of a venta. PaidAt is RFC 3339; empty means now.
type RegisterPaymentRequest struct {
	InstallmentIDs     []string    `json:"cuotaIds"`
	IncludeDownPayment bool        `json:"incluirEntrega"`
	Charges            []ChargeDTO `json:"cargos,omitempty"`
	Medium             string      `json:"medioPago"`
	PaidAt             string      `json:"fechaPago,omitempty"`
	Observation        string      `json:"observacion,omitempty"`
}

// SettleSaleRequest collects everything still owed. ExpectedAmount is the
// saldo the caller saw; it is required, and the backend rejects the cobro
// when it no longer matches.
type SettleSaleRequest struct {
	ExpectedAmount *float64    `json:"montoEsperado"`
	Charges        []ChargeDTO `json:"cargos,omitempty"`
	Medium         string      `json:"medioPago"`
	PaidAt         string      `json:"fechaPago,omitempty"`
	Observation    string      `json:"observacion,omitempty"`
}
