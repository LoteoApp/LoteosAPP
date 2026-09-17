package gateway

import "context"

// IdentityProvider abstracts the identity provider so the business layer
// creates and removes accounts without depending on a concrete adapter.
type IdentityProvider interface {
	// CreateUser creates an account for email with the given rol, without a
	// password, and returns its identity provider ID plus a one-time invite
	// link the recipient uses to set their own password. It returns
	// domain.ErrEmailEnUso if the email is already registered.
	CreateUser(ctx context.Context, email, rol string) (authProviderID, inviteURL string, err error)

	// DeleteUser removes the account. Used to compensate a CreateUser call
	// whose local persistence step failed afterwards.
	DeleteUser(ctx context.Context, authProviderID string) error

	// GenerateInviteLink mints a fresh one-time invite link for an existing,
	// still-unconfirmed account. Used to retry an invite email: the original
	// link is never persisted, so a retry mints a new one instead of
	// resending a link that only ever lived in memory. It operates by email
	// rather than authProviderID because that's what the identity provider's
	// link-generation call takes.
	GenerateInviteLink(ctx context.Context, email string) (inviteURL string, err error)

	// SetPassword sets newPassword on an existing account, chosen by the
	// account's own owner. Used to confirm a "forgot password" request once
	// its token has been verified.
	SetPassword(ctx context.Context, authProviderID, newPassword string) error
}
