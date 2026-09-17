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

	ResetPasswordCalls  int
	ResetPasswordErr    error
	ResetPasswordUserID string
	ResetPasswordResult string

	SetPasswordCalls    int
	SetPasswordErr      error
	SetPasswordUserID   string
	SetPasswordPassword string
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

func (fake *IdentityProvider) ResetTemporaryPassword(_ context.Context, authProviderID string) (string, error) {
	fake.ResetPasswordCalls++
	fake.ResetPasswordUserID = authProviderID
	if fake.ResetPasswordErr != nil {
		return "", fake.ResetPasswordErr
	}
	return fake.ResetPasswordResult, nil
}

func (fake *IdentityProvider) SetPassword(_ context.Context, authProviderID, newPassword string) error {
	fake.SetPasswordCalls++
	fake.SetPasswordUserID = authProviderID
	fake.SetPasswordPassword = newPassword
	return fake.SetPasswordErr
}
