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
}
