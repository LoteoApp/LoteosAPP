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

	ConfirmedAccountIDsCalls  int
	ConfirmedAccountIDsErr    error
	ConfirmedAccountIDsResult map[string]bool
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

func (fake *IdentityProvider) ConfirmedAccountIDs(_ context.Context) (map[string]bool, error) {
	fake.ConfirmedAccountIDsCalls++
	if fake.ConfirmedAccountIDsErr != nil {
		return nil, fake.ConfirmedAccountIDsErr
	}
	return fake.ConfirmedAccountIDsResult, nil
}
