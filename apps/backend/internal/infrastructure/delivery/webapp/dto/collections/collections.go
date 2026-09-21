package dto

// RegisterPaymentRequest collects the entrega and/or the next pending
// cuotas of a venta. FechaPago is RFC 3339; empty means now.
type RegisterPaymentRequest struct {
	CuotaIDs       []string `json:"cuotaIds"`
	IncluirEntrega bool     `json:"incluirEntrega"`
	MedioPago      string   `json:"medioPago"`
	FechaPago      string   `json:"fechaPago,omitempty"`
	Observacion    string   `json:"observacion,omitempty"`
}

// SettleSaleRequest collects everything still owed. MontoEsperado is the
// saldo the caller saw; the backend rejects the cobro when it no longer
// matches. nil skips the check.
type SettleSaleRequest struct {
	MontoEsperado *float64 `json:"montoEsperado,omitempty"`
	MedioPago     string   `json:"medioPago"`
	FechaPago     string   `json:"fechaPago,omitempty"`
	Observacion   string   `json:"observacion,omitempty"`
}
