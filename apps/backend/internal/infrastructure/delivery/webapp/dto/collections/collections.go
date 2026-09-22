package dto

// ChargeDTO is a cargo adicional collected with a cobro (impuestos, gastos
// administrativos, honorarios, servicios...). Moneda es opcional: vacía
// significa la moneda de la venta, y una distinta se cobra aparte sin
// convertirse.
type ChargeDTO struct {
	Tipo    string  `json:"tipo"`
	Monto   float64 `json:"monto"`
	Moneda  string  `json:"moneda,omitempty"`
	Detalle string  `json:"detalle,omitempty"`
}

// RegisterPaymentRequest collects the entrega and/or the next pending
// cuotas of a venta. FechaPago is RFC 3339; empty means now.
type RegisterPaymentRequest struct {
	CuotaIDs       []string    `json:"cuotaIds"`
	IncluirEntrega bool        `json:"incluirEntrega"`
	Cargos         []ChargeDTO `json:"cargos,omitempty"`
	MedioPago      string      `json:"medioPago"`
	FechaPago      string      `json:"fechaPago,omitempty"`
	Observacion    string      `json:"observacion,omitempty"`
}

// SettleSaleRequest collects everything still owed. MontoEsperado is the
// saldo the caller saw; the backend rejects the cobro when it no longer
// matches. nil skips the check.
type SettleSaleRequest struct {
	MontoEsperado *float64    `json:"montoEsperado,omitempty"`
	Cargos        []ChargeDTO `json:"cargos,omitempty"`
	MedioPago     string      `json:"medioPago"`
	FechaPago     string      `json:"fechaPago,omitempty"`
	Observacion   string      `json:"observacion,omitempty"`
}
