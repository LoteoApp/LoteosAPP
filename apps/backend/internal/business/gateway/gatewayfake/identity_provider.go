package gatewayfake

import "context"

// IdentityProvider is a fake gateway.IdentityProvider for tests.
type IdentityProvider struct {
	CreateCalls    int
	CreateErr      error
	AuthProviderID string
	TempPassword   string

	DeleteCalls   int
	DeleteErr     error
	DeletedUserID string
}

func (fake *IdentityProvider) CreateUser(_ context.Context, email, rol string) (string, string, error) {
	fake.CreateCalls++
	if fake.CreateErr != nil {
		return "", "", fake.CreateErr
	}
	return fake.AuthProviderID, fake.TempPassword, nil
}

func (fake *IdentityProvider) DeleteUser(_ context.Context, authProviderID string) error {
	fake.DeleteCalls++
	fake.DeletedUserID = authProviderID
	return fake.DeleteErr
}
