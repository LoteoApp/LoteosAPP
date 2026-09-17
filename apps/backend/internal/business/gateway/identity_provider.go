package gateway

import "context"

// IdentityProvider abstracts the identity provider so the business layer
// creates and removes accounts without depending on a concrete adapter.
type IdentityProvider interface {
	// CreateUser creates an account for email with the given rol and returns
	// its identity provider ID plus a one-time temporary password. It
	// returns domain.ErrEmailEnUso if the email is already registered.
	CreateUser(ctx context.Context, email, rol string) (authProviderID, temporaryPassword string, err error)

	// DeleteUser removes the account. Used to compensate a CreateUser call
	// whose local persistence step failed afterwards.
	DeleteUser(ctx context.Context, authProviderID string) error

	// ResetTemporaryPassword sets a fresh one-time password on an existing
	// account and returns it. Used to retry an invite email: the original
	// temporary password is never persisted, so a retry mints a new one
	// instead of resending a password that only ever lived in memory.
	ResetTemporaryPassword(ctx context.Context, authProviderID string) (temporaryPassword string, err error)

	// SetPassword sets newPassword on an existing account, chosen by the
	// account's own owner (unlike ResetTemporaryPassword, which mints a
	// random one). Used to confirm a "forgot password" request once its
	// token has been verified.
	SetPassword(ctx context.Context, authProviderID, newPassword string) error
}
