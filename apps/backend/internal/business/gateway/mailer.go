package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

// UserInviteEmail is the invite sent to a user right after their account is
// created (or, on retry, after a fresh invite link is minted for them).
// InviteURL is a one-time link the recipient opens to choose their own
// password; no password ever travels in this email.
type UserInviteEmail struct {
	To               string
	Nombre, Apellido string
	Rol              domain.Rol
	InviteURL        string
}

// PasswordResetEmail carries the link a "forgot password" request sends. The
// link itself, not a new password, is what goes out: the password only
// changes once the recipient opens it and confirms one of their own choosing.
type PasswordResetEmail struct {
	To               string
	Nombre, Apellido string
	ResetURL         string
}

// Mailer abstracts the outbound email provider so the business layer sends
// mail without depending on a concrete adapter.
type Mailer interface {
	SendUserInvite(ctx context.Context, invite UserInviteEmail) error
	SendPasswordReset(ctx context.Context, reset PasswordResetEmail) error
}
