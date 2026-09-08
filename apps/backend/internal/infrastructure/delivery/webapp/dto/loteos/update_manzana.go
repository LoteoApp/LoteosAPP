package dto

type UpdateManzanaRequest struct {
	Number   string   `json:"numero"`
	HasWater bool     `json:"tieneAgua"`
	HasSewer bool     `json:"tieneCloaca"`
	HasPower bool     `json:"tieneLuz"`
	HasGas   bool     `json:"tieneGas"`
	CalleIDs []string `json:"calleIds"`
}
