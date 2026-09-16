package dto

// UpdateLoteRequest carries the values a lot only gets by hand, since the DXF
// layers hold geometry with no text. Price and Area are pointers so an
// omitted field clears the column instead of writing a zero.
type UpdateLoteRequest struct {
	Number   string   `json:"numero"`
	Price    *float64 `json:"precio"`
	Currency string   `json:"moneda"`
	Area     *float64 `json:"superficie"`
	Features string   `json:"caracteristicas"`
}
