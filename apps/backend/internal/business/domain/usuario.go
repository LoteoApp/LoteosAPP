package domain

import (
	"strings"
	"time"
)

var (
	ErrEmailEnUso          = &Error{Kind: KindConflict, Code: "email_in_use", Message: "El email ya está en uso"}
	ErrUsuarioNoEncontrado = &Error{Kind: KindNotFound, Code: "user_not_found", Message: "Usuario no encontrado"}
	ErrNoAutorizado        = &Error{Kind: KindForbidden, Code: "forbidden", Message: "No tenés permisos para esta acción"}
	ErrEmailInvalido       = &Error{Kind: KindInvalid, Code: "invalid_email", Message: "Email inválido"}
	ErrRolInvalido         = &Error{Kind: KindInvalid, Code: "invalid_rol", Message: "Rol inválido"}
	ErrPerfilInvalido      = &Error{Kind: KindInvalid, Code: "invalid_profile", Message: "Nombre y apellido son obligatorios"}
	// ErrActorNoAprovisionado: the token is valid but its subject has no row
	// in usuarios, so there is no local user to attribute the operation to.
	ErrActorNoAprovisionado = &Error{Kind: KindForbidden, Code: "actor_not_provisioned", Message: "Tu usuario no está habilitado para operar en el sistema"}
	ErrUsuarioIDInvalido    = &Error{Kind: KindInvalid, Code: "invalid_user_id", Message: "Identificador de usuario inválido"}
	ErrUsuarioDadoDeBaja    = &Error{Kind: KindConflict, Code: "user_already_inactive", Message: "El usuario ya está dado de baja"}
	ErrUsuarioSinCambios    = &Error{Kind: KindInvalid, Code: "empty_user_update", Message: "No se enviaron campos para modificar"}
	ErrUsuarioYaActivo      = &Error{Kind: KindConflict, Code: "user_already_active", Message: "El usuario ya está activo"}
	// ErrCuentaInactiva: the caller's token is valid and its subject has a
	// usuarios row, but that row is given de baja. Distinct from
	// ErrActorNoAprovisionado (no row at all) and from ErrUsuarioDadoDeBaja
	// (a write targeting an inactive user) — this one blocks every request
	// from an inactive caller, checked once per request in middleware.
	ErrCuentaInactiva = &Error{Kind: KindForbidden, Code: "account_inactive", Message: "Tu cuenta fue dada de baja"}
)

type Usuario struct {
	ID             string     `json:"id"`
	AuthProviderID string     `json:"-"`
	Email          string     `json:"email"`
	Nombre         string     `json:"nombre"`
	Apellido       string     `json:"apellido"`
	Rol            Rol        `json:"rol"`
	PerfilCompleto bool       `json:"perfilCompleto"`
	FechaBaja      *time.Time `json:"fechaBaja"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// Activo reports whether the user may still operate. A user given de baja
// keeps its row so the audit foreign keys that point at it
// (usuario_modificacion, usuario_loteos, reservas, ventas) stay valid.
func (usuario Usuario) Activo() bool {
	return usuario.FechaBaja == nil
}

// UsuarioUpdate is a partial change to a user: a nil field is left
// unchanged. Email and Rol aren't part of it — email identifies the account
// in the identity provider, and the role is fixed at creation; both are
// separate operations this ABM doesn't support yet.
type UsuarioUpdate struct {
	ID                  string
	Nombre              *string
	Apellido            *string
	UsuarioModificacion string
}

func EmailValido(email string) bool {
	return email != "" && strings.Contains(email, "@")
}
