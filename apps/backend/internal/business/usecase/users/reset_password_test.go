package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestResetPasswordHappyPath(t *testing.T) {
	t.Parallel()

	usuario := registeredActiveUser()
	usuario.AuthProviderID = "sb-123"
	repository := &gatewayfake.UserRepository{FoundByID: usuario}
	identity := &gatewayfake.IdentityProvider{}
	tokens := NewPasswordResetTokens()
	now := time.Now()
	token, err := tokens.Issue(usuario.ID, now, PasswordResetTokenTTL)
	if err != nil {
		t.Fatalf("tokens.Issue() error = %v", err)
	}
	resetPassword := NewResetPassword(repository, identity, tokens, fixedClock{now: now})

	if err := resetPassword.Execute(context.Background(), token, "a-new-password"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if identity.SetPasswordCalls != 1 || identity.SetPasswordUserID != "sb-123" || identity.SetPasswordPassword != "a-new-password" {
		t.Errorf("Execute() identity.SetPassword calls = %d, userID = %q, password = %q",
			identity.SetPasswordCalls, identity.SetPasswordUserID, identity.SetPasswordPassword)
	}
}

func TestResetPasswordRejectsShortPassword(t *testing.T) {
	t.Parallel()

	identity := &gatewayfake.IdentityProvider{}
	resetPassword := NewResetPassword(&gatewayfake.UserRepository{}, identity, NewPasswordResetTokens())

	err := resetPassword.Execute(context.Background(), "any-token", "short")

	if !errors.Is(err, domain.ErrPasswordInvalido) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrPasswordInvalido)
	}
	if identity.SetPasswordCalls != 0 {
		t.Error("Execute() should not touch the token or the identity provider for an invalid password")
	}
}

func TestResetPasswordRejectsUnknownToken(t *testing.T) {
	t.Parallel()

	identity := &gatewayfake.IdentityProvider{}
	resetPassword := NewResetPassword(&gatewayfake.UserRepository{}, identity, NewPasswordResetTokens())

	err := resetPassword.Execute(context.Background(), "does-not-exist", "a-new-password")

	if !errors.Is(err, domain.ErrPasswordResetTokenInvalido) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrPasswordResetTokenInvalido)
	}
	if identity.SetPasswordCalls != 0 {
		t.Error("Execute() should not call the identity provider for an invalid token")
	}
}

func TestResetPasswordRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	tokens := NewPasswordResetTokens()
	now := time.Now()
	token, err := tokens.Issue("user-1", now, PasswordResetTokenTTL)
	if err != nil {
		t.Fatalf("tokens.Issue() error = %v", err)
	}
	resetPassword := NewResetPassword(&gatewayfake.UserRepository{}, &gatewayfake.IdentityProvider{}, tokens,
		fixedClock{now: now.Add(PasswordResetTokenTTL + time.Second)})

	err = resetPassword.Execute(context.Background(), token, "a-new-password")

	if !errors.Is(err, domain.ErrPasswordResetTokenInvalido) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrPasswordResetTokenInvalido)
	}
}

func TestResetPasswordTokenIsSingleUse(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: registeredActiveUser()}
	tokens := NewPasswordResetTokens()
	now := time.Now()
	token, err := tokens.Issue("user-1", now, PasswordResetTokenTTL)
	if err != nil {
		t.Fatalf("tokens.Issue() error = %v", err)
	}
	resetPassword := NewResetPassword(repository, &gatewayfake.IdentityProvider{}, tokens, fixedClock{now: now})

	if err := resetPassword.Execute(context.Background(), token, "a-new-password"); err != nil {
		t.Fatalf("Execute() first call error = %v", err)
	}

	err = resetPassword.Execute(context.Background(), token, "another-password")

	if !errors.Is(err, domain.ErrPasswordResetTokenInvalido) {
		t.Fatalf("Execute() second call error = %v, want %v — a token must not be redeemable twice", err, domain.ErrPasswordResetTokenInvalido)
	}
}

func TestResetPasswordRejectsInactiveUser(t *testing.T) {
	t.Parallel()

	baja := time.Now()
	inactive := registeredActiveUser()
	inactive.FechaBaja = &baja
	repository := &gatewayfake.UserRepository{FoundByID: inactive}
	identity := &gatewayfake.IdentityProvider{}
	tokens := NewPasswordResetTokens()
	now := time.Now()
	token, err := tokens.Issue(inactive.ID, now, PasswordResetTokenTTL)
	if err != nil {
		t.Fatalf("tokens.Issue() error = %v", err)
	}
	resetPassword := NewResetPassword(repository, identity, tokens, fixedClock{now: now})

	err = resetPassword.Execute(context.Background(), token, "a-new-password")

	if !errors.Is(err, domain.ErrUsuarioDadoDeBaja) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioDadoDeBaja)
	}
	if identity.SetPasswordCalls != 0 {
		t.Error("Execute() should not set a password for an inactive account")
	}
}

func TestResetPasswordPropagatesSetPasswordFailure(t *testing.T) {
	t.Parallel()

	cause := errors.New("supabase unavailable")
	repository := &gatewayfake.UserRepository{FoundByID: registeredActiveUser()}
	identity := &gatewayfake.IdentityProvider{SetPasswordErr: cause}
	tokens := NewPasswordResetTokens()
	now := time.Now()
	token, err := tokens.Issue("user-1", now, PasswordResetTokenTTL)
	if err != nil {
		t.Fatalf("tokens.Issue() error = %v", err)
	}
	resetPassword := NewResetPassword(repository, identity, tokens, fixedClock{now: now})

	err = resetPassword.Execute(context.Background(), token, "a-new-password")

	assertDatabaseUnavailable(t, err, cause)
}
