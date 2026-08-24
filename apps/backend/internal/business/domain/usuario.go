package domain

import "time"

var (
	ErrEmailEnUso          = &Error{Kind: KindConflict, Code: "email_in_use", Message: "El email ya está en uso"}
	ErrUsuarioNoEncontrado = &Error{Kind: KindNotFound, Code: "user_not_found", Message: "Usuario no encontrado"}
	ErrNoAutorizado        = &Error{Kind: KindForbidden, Code: "forbidden", Message: "No tenés permisos para esta acción"}
	ErrEmailInvalido       = &Error{Kind: KindInvalid, Code: "invalid_email", Message: "Email inválido"}
	ErrRolInvalido         = &Error{Kind: KindInvalid, Code: "invalid_rol", Message: "Rol inválido"}
	ErrPerfilInvalido      = &Error{Kind: KindInvalid, Code: "invalid_profile", Message: "Nombre y apellido son obligatorios"}
)

type Usuario struct {
	ID             string    `json:"id"`
	AuthProviderID string    `json:"-"`
	Email          string    `json:"email"`
	Nombre         string    `json:"nombre"`
	Apellido       string    `json:"apellido"`
	Rol            Rol       `json:"rol"`
	PerfilCompleto bool      `json:"perfilCompleto"`
	CreatedAt      time.Time `json:"createdAt"`
}
