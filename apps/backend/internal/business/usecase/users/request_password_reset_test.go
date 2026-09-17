package users

import (
	"context"
	"fmt"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

const testResetURL = "https://app.loteosapp.com/restablecer-contrasena"

type fixedClock struct{ now time.Time }

func (clock fixedClock) Now() time.Time { return clock.now }

type mutableClock struct{ now time.Time }

func (clock *mutableClock) Now() time.Time { return clock.now }

func registeredActiveUser() domain.Usuario {
	usuario := activeManagedUserWithContact()
	usuario.ID = "user-1"
	return usuario
}

func TestRequestPasswordResetSendsEmailForRegisteredUser(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByEmail: registeredActiveUser()}
	mailer := &gatewayfake.Mailer{}
	tokens := NewPasswordResetTokens()
	requestReset := NewRequestPasswordReset(repository, mailer, tokens, testResetURL, fixedClock{now: time.Now()})

	if err := requestReset.Execute(context.Background(), "ana@example.com"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if repository.FindByEmailInput != "ana@example.com" {
		t.Errorf("Execute() looked up %q, want %q", repository.FindByEmailInput, "ana@example.com")
	}
	if mailer.SendPasswordResetCalls != 1 {
		t.Fatalf("Execute() mailer.SendPasswordReset calls = %d, want 1", mailer.SendPasswordResetCalls)
	}
	sent := mailer.SendPasswordResetInputs[0]
	if sent.To != "ana@example.com" || sent.ResetURL == testResetURL {
		t.Errorf("Execute() sent reset = %#v, want a ResetURL with a token appended", sent)
	}
}

func TestRequestPasswordResetSucceedsSilentlyForUnknownEmail(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByEmailErr: domain.ErrUsuarioNoEncontrado}
	mailer := &gatewayfake.Mailer{}
	requestReset := NewRequestPasswordReset(repository, mailer, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()})

	if err := requestReset.Execute(context.Background(), "unknown@example.com"); err != nil {
		t.Fatalf("Execute() error = %v, want nil so the response can't be used to tell registered emails apart", err)
	}
	if mailer.SendPasswordResetCalls != 0 {
		t.Error("Execute() should not send an email for an unknown address")
	}
}

func TestRequestPasswordResetSucceedsSilentlyForInactiveUser(t *testing.T) {
	t.Parallel()

	baja := time.Now()
	inactive := registeredActiveUser()
	inactive.FechaBaja = &baja
	repository := &gatewayfake.UserRepository{FoundByEmail: inactive}
	mailer := &gatewayfake.Mailer{}
	requestReset := NewRequestPasswordReset(repository, mailer, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()})

	if err := requestReset.Execute(context.Background(), "ana@example.com"); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if mailer.SendPasswordResetCalls != 0 {
		t.Error("Execute() should not send an email for an inactive account")
	}
}

func TestRequestPasswordResetRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	requestReset := NewRequestPasswordReset(&gatewayfake.UserRepository{}, &gatewayfake.Mailer{}, NewPasswordResetTokens(), testResetURL)

	err := requestReset.Execute(context.Background(), "not-an-email")

	if err != domain.ErrEmailInvalido {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailInvalido)
	}
}

func TestRequestPasswordResetRejectsWithinCooldown(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByEmail: registeredActiveUser()}
	mailer := &gatewayfake.Mailer{}
	requestReset := NewRequestPasswordReset(repository, mailer, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()})

	if err := requestReset.Execute(context.Background(), "ana@example.com"); err != nil {
		t.Fatalf("Execute() first call error = %v", err)
	}

	err := requestReset.Execute(context.Background(), "ana@example.com")

	if err != domain.ErrPasswordResetRateLimited {
		t.Fatalf("Execute() second call error = %v, want %v", err, domain.ErrPasswordResetRateLimited)
	}
	if mailer.SendPasswordResetCalls != 1 {
		t.Errorf("Execute() mailer.SendPasswordReset calls = %d, want 1", mailer.SendPasswordResetCalls)
	}
}

func TestRequestPasswordResetCooldownAppliesEvenForUnknownEmail(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByEmailErr: domain.ErrUsuarioNoEncontrado}
	requestReset := NewRequestPasswordReset(repository, &gatewayfake.Mailer{}, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()})

	if err := requestReset.Execute(context.Background(), "unknown@example.com"); err != nil {
		t.Fatalf("Execute() first call error = %v", err)
	}

	err := requestReset.Execute(context.Background(), "unknown@example.com")

	if err != domain.ErrPasswordResetRateLimited {
		t.Fatalf("Execute() second call error = %v, want %v — the cooldown must not depend on the email existing", err, domain.ErrPasswordResetRateLimited)
	}
}

// TestRequestPasswordResetSweepsExpiredCooldownEntries guards against
// unbounded growth on an unauthenticated endpoint: without a sweep, an
// attacker could send a different made-up email on every request and each
// one would sit in lastSent forever, since this cooldown is checked before
// the email is even looked up.
func TestRequestPasswordResetSweepsExpiredCooldownEntries(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByEmailErr: domain.ErrUsuarioNoEncontrado}
	clock := &mutableClock{now: time.Now()}
	requestReset := NewRequestPasswordReset(repository, &gatewayfake.Mailer{}, NewPasswordResetTokens(), testResetURL, clock)
	useCase := requestReset.(*requestPasswordResetUseCase)

	for i := 0; i < 5; i++ {
		email := fmt.Sprintf("made-up-%d@example.com", i)
		if err := requestReset.Execute(context.Background(), email); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	}
	if len(useCase.lastSent) != 5 {
		t.Fatalf("len(lastSent) = %d, want 5 before the cooldown elapses", len(useCase.lastSent))
	}

	clock.now = clock.now.Add(passwordResetRequestCooldown)
	if err := requestReset.Execute(context.Background(), "yet-another@example.com"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(useCase.lastSent) != 1 {
		t.Errorf("len(lastSent) = %d, want 1 — the sweep should have dropped the 5 stale entries", len(useCase.lastSent))
	}
}

func TestRequestPasswordResetAllowsAfterCooldownElapses(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByEmail: registeredActiveUser()}
	mailer := &gatewayfake.Mailer{}
	clock := &mutableClock{now: time.Now()}
	requestReset := NewRequestPasswordReset(repository, mailer, NewPasswordResetTokens(), testResetURL, clock)

	if err := requestReset.Execute(context.Background(), "ana@example.com"); err != nil {
		t.Fatalf("Execute() first call error = %v", err)
	}

	clock.now = clock.now.Add(passwordResetRequestCooldown)

	if err := requestReset.Execute(context.Background(), "ana@example.com"); err != nil {
		t.Fatalf("Execute() call after cooldown error = %v", err)
	}
	if mailer.SendPasswordResetCalls != 2 {
		t.Errorf("Execute() mailer.SendPasswordReset calls = %d, want 2", mailer.SendPasswordResetCalls)
	}
}
