package gatewayfake

import "context"

// IdentityProvider is a fake gateway.IdentityProvider for tests.
type IdentityProvider struct {
	CreateCalls    int
	CreateErr      error
	AuthProviderID string
	InviteURL      string

	DeleteCalls   int
	DeleteErr     error
	DeletedUserID string

	GenerateInviteLinkCalls int
	GenerateInviteLinkErr   error
	GenerateInviteLinkEmail string
	GenerateInviteLinkURL   string
}

func (fake *IdentityProvider) CreateUser(_ context.Context, email, rol string) (string, string, error) {
	fake.CreateCalls++
	if fake.CreateErr != nil {
		return "", "", fake.CreateErr
	}
	return fake.AuthProviderID, fake.InviteURL, nil
}

func (fake *IdentityProvider) DeleteUser(_ context.Context, authProviderID string) error {
	fake.DeleteCalls++
	fake.DeletedUserID = authProviderID
	return fake.DeleteErr
}

func (fake *IdentityProvider) GenerateInviteLink(_ context.Context, email string) (string, error) {
	fake.GenerateInviteLinkCalls++
	fake.GenerateInviteLinkEmail = email
	if fake.GenerateInviteLinkErr != nil {
		return "", fake.GenerateInviteLinkErr
	}
	return fake.GenerateInviteLinkURL, nil
}
