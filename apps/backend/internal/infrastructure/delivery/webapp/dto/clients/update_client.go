package dto

// UpdateClientRequest is a partial update: every field is optional. A field
// omitted from the JSON body (nil after decoding) leaves the stored value
// unchanged, which is what PATCH /api/v1/clientes/{id} is expected to do —
// see clients.UpdateClient.
type UpdateClientRequest struct {
	Nombre   *string `json:"nombre,omitempty"`
	Apellido *string `json:"apellido,omitempty"`
	DNI      *string `json:"dni,omitempty"`
	Celular  *string `json:"celular,omitempty"`
	Email    *string `json:"email,omitempty"`
}
