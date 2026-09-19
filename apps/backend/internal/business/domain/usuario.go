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
	// ErrAgenciaRequerida: a user with rol inmobiliaria must belong to an
	// agency, so this ABM requires one at creation time.
	ErrAgenciaRequerida = &Error{Kind: KindInvalid, Code: "agency_required", Message: "Seleccioná la inmobiliaria a la que pertenece el usuario"}
	// ErrCuentaInactiva: the caller's token is valid and its subject has a
	// usuarios row, but that row is given de baja. Distinct from
	// ErrActorNoAprovisionado (no row at all) and from ErrUsuarioDadoDeBaja
	// (a write targeting an inactive user) — this one blocks every request
	// from an inactive caller, checked once per request in middleware.
	ErrCuentaInactiva = &Error{Kind: KindForbidden, Code: "account_inactive", Message: "Tu cuenta fue dada de baja"}
	// ErrInviteEmailUnavailable: ResendInviteEmail's send failed. Unlike
	// CreateUser (which swallows the same failure — the admin already has
	// the temporary password to hand over another way), this is a caller
	// explicitly asking for the mail to go out, so it's surfaced instead of
	// logged and dropped.
	ErrInviteEmailUnavailable = &Error{Kind: KindUnavailable, Code: "invite_email_unavailable", Message: "No se pudo enviar el mail de invitación"}
	// ErrInviteEmailRateLimited: ResendInviteEmail was called again for the
	// same user before its in-memory cooldown elapsed. Protects the mail
	// provider's quota from a double click, a network retry, or several
	// admin tabs open at once.
	ErrInviteEmailRateLimited = &Error{Kind: KindRateLimited, Code: "invite_email_rate_limited", Message: "Esperá un momento antes de volver a reenviar la invitación"}
	// ErrPasswordResetRateLimited: RequestPasswordReset was called again for
	// the same email before its in-memory cooldown elapsed. Checked before
	// looking the email up, so it applies the same way whether or not the
	// email belongs to a real account — that's what keeps the cooldown from
	// doubling as an existence check.
	ErrPasswordResetRateLimited = &Error{Kind: KindRateLimited, Code: "password_reset_rate_limited", Message: "Ya te enviamos un mail. Esperá un momento antes de pedir otro"}
	// ErrPasswordResetBusy: RequestPasswordReset already has its maximum of
	// background jobs in flight. Depends only on load, never on the email, so
	// it can't be used to tell registered emails apart.
	ErrPasswordResetBusy = &Error{Kind: KindUnavailable, Code: "password_reset_busy", Message: "Estamos recibiendo muchos pedidos. Probá de nuevo en unos minutos"}
	// ErrPasswordResetTokenInvalido: ResetPassword got a token that's
	// unknown, expired, or already used. All three collapse into the same
	// error and message — telling them apart would let a caller probe for
	// which tokens once existed.
	ErrPasswordResetTokenInvalido = &Error{Kind: KindInvalid, Code: "password_reset_token_invalid", Message: "El link para restablecer la contraseña es inválido o venció"}
	ErrPasswordInvalido           = &Error{Kind: KindInvalid, Code: "invalid_password", Message: "La contraseña debe tener al menos 8 caracteres"}
	// ErrInviteAlreadyAccepted: ResendInviteEmail targeted an account that
	// already confirmed itself, for which the identity provider refuses to
	// mint another invite link.
	ErrInviteAlreadyAccepted = &Error{Kind: KindConflict, Code: "invite_already_accepted", Message: "El usuario ya activó su cuenta, no necesita otra invitación"}
)

type Usuario struct {
	ID             string `json:"id"`
	AuthProviderID string `json:"-"`
	Email          string `json:"email"`
	Nombre         string `json:"nombre"`
	Apellido       string `json:"apellido"`
	Rol            Rol    `json:"rol"`
	// AgencyID is only set for rol inmobiliaria: the agency this user
	// operates on behalf of. Nil for every other role.
	AgencyID       *string    `json:"inmobiliariaId,omitempty"`
	PerfilCompleto bool       `json:"perfilCompleto"`
	FechaBaja      *time.Time `json:"fechaBaja"`
	CreatedAt      time.Time  `json:"createdAt"`
	// InvitacionAceptada is only set where the identity provider was asked:
	// nil means unknown, not "pending".
	InvitacionAceptada *bool `json:"invitacionAceptada,omitempty"`
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

// maxEmailLength follows RFC 5321's 254-character limit on the reverse-path,
// which in practice bounds a full email address. Enforcing it here also
// keeps an unauthenticated caller (password reset) from making the JSON
// decoder and every downstream string operation work on an arbitrarily
// large value.
const maxEmailLength = 254

func EmailValido(email string) bool {
	return email != "" && len(email) <= maxEmailLength && strings.Contains(email, "@")
}
