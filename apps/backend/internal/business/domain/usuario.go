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

	ErrAgrimensorIDInvalido   = &Error{Kind: KindInvalid, Code: "invalid_surveyor_id", Message: "Identificador de agrimensor inválido"}
	ErrAgrimensorNoEncontrado = &Error{Kind: KindNotFound, Code: "surveyor_not_found", Message: "Agrimensor no encontrado"}
	ErrAgrimensorDadoDeBaja   = &Error{Kind: KindConflict, Code: "surveyor_already_inactive", Message: "El agrimensor ya está dado de baja"}
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

func (usuario Usuario) EsAgrimensor() bool {
	return usuario.Rol == RolAgrimensor
}

// PerfilEstaCompleto reports whether nombre and apellido are both filled in,
// which is what "perfil completo" means for a user the administrador creates
// with a full name instead of leaving it for the user to complete.
func PerfilEstaCompleto(nombre, apellido string) bool {
	return nombre != "" && apellido != ""
}

// UsuarioUpdate is a partial change to a user: a nil field is left unchanged.
// Email is not part of it because it identifies the account in the identity
// provider, which this update does not touch.
type UsuarioUpdate struct {
	ID                  string
	Nombre              *string
	Apellido            *string
	UsuarioModificacion string
}

func EmailValido(email string) bool {
	return email != "" && strings.Contains(email, "@")
}
