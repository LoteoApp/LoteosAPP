package gateway

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

// UserInviteEmail is the invite sent to a user right after their account is
// created (or, on retry, after a fresh temporary password is minted for
// them). LoginURL points at the frontend's login page.
type UserInviteEmail struct {
	To                string
	Nombre, Apellido  string
	Rol               domain.Rol
	TemporaryPassword string
	LoginURL          string
}

// Mailer abstracts the outbound email provider so the business layer sends
// mail without depending on a concrete adapter.
type Mailer interface {
	SendUserInvite(ctx context.Context, invite UserInviteEmail) error
}
