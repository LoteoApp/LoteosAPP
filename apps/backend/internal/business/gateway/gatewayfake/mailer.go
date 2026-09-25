package gatewayfake

import (
	"context"

	"loteosapp/backend/internal/business/gateway"
)

// Mailer is a fake gateway.Mailer for tests. SendUserInviteErrs, if set, is
// consumed one error per call (nil once exhausted) so a test can make the
// Nth send in a batch fail without affecting the others.
type Mailer struct {
	SendUserInviteCalls  int
	SendUserInviteErr    error
	SendUserInviteErrs   []error
	SendUserInviteInputs []gateway.UserInviteEmail

	SendPasswordResetCalls  int
	SendPasswordResetErr    error
	SendPasswordResetInputs []gateway.PasswordResetEmail
}

func (fake *Mailer) SendUserInvite(_ context.Context, invite gateway.UserInviteEmail) error {
	index := fake.SendUserInviteCalls
	fake.SendUserInviteCalls++
	fake.SendUserInviteInputs = append(fake.SendUserInviteInputs, invite)

	if index < len(fake.SendUserInviteErrs) {
		return fake.SendUserInviteErrs[index]
	}
	return fake.SendUserInviteErr
}

func (fake *Mailer) SendPasswordReset(_ context.Context, reset gateway.PasswordResetEmail) error {
	fake.SendPasswordResetCalls++
	fake.SendPasswordResetInputs = append(fake.SendPasswordResetInputs, reset)
	return fake.SendPasswordResetErr
}
