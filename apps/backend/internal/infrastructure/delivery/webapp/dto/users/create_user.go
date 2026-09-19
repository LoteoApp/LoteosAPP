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
	// InviteEmailSent is false when the best-effort invite email failed to
	// go out; the usuario has no password yet either way, and
	// POST /api/v1/usuarios/{id}/reenviar-invitacion retries the email.
	InviteEmailSent bool `json:"invitacionEnviada"`
}
