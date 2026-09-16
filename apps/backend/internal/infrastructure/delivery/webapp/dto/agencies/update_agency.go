package dto

// UpdateAgencyRequest is a partial update: every field is optional. A field
// omitted from the JSON body (nil after decoding) leaves the stored value
// unchanged, which is what PATCH /api/v1/inmobiliarias/{id} is expected to
// do — see agencies.UpdateAgency, which also rejects a body that carries no
// field at all.
type UpdateAgencyRequest struct {
	BusinessName *string `json:"razonSocial,omitempty"`
	CUIT         *string `json:"cuit,omitempty"`
	Phone        *string `json:"telefono,omitempty"`
	Email        *string `json:"email,omitempty"`
}
