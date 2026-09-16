package dto

import "loteosapp/backend/internal/business/domain"

type CreateUserRequest struct {
	Nombre         string `json:"nombre"`
	Apellido       string `json:"apellido"`
	Email          string `json:"email"`
	Rol            string `json:"rol"`
	InmobiliariaID string `json:"inmobiliariaId"`
}

type CreateUserResponse struct {
	domain.Usuario
	TemporaryPassword string `json:"temporaryPassword"`
	// InviteEmailSent is false when the best-effort invite email failed to
	// go out; the temporary password above still works, and
	// POST /api/v1/usuarios/{id}/reenviar-invitacion retries the email.
	InviteEmailSent bool `json:"invitacionEnviada"`
}
