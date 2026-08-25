package dto

type UpdateClientRequest struct {
	Nombre   string  `json:"nombre"`
	Apellido string  `json:"apellido"`
	DNI      string  `json:"dni"`
	Celular  *string `json:"celular,omitempty"`
	Email    *string `json:"email,omitempty"`
}
