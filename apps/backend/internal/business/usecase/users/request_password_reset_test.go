package users

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
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

// newSyncRequestPasswordReset builds a RequestPasswordReset whose dispatched
// work runs inline instead of in a goroutine, so a test can assert its
// effects right after Execute returns without racing a background send.
func newSyncRequestPasswordReset(
	repository gateway.UserRepository,
	mailer gateway.Mailer,
	tokens *PasswordResetTokens,
	resetURL string,
	clocks ...Clock,
) *requestPasswordResetUseCase {
	useCase := NewRequestPasswordReset(repository, mailer, tokens, resetURL, clocks...).(*requestPasswordResetUseCase)
	useCase.dispatch = func(work func()) { work() }
	return useCase
}

func TestRequestPasswordResetSendsEmailForRegisteredUser(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByEmail: registeredActiveUser()}
	mailer := &gatewayfake.Mailer{}
	tokens := NewPasswordResetTokens()
	requestReset := newSyncRequestPasswordReset(repository, mailer, tokens, testResetURL, fixedClock{now: time.Now()})

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
	requestReset := newSyncRequestPasswordReset(repository, mailer, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()})

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
	requestReset := newSyncRequestPasswordReset(repository, mailer, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()})

	if err := requestReset.Execute(context.Background(), "ana@example.com"); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if mailer.SendPasswordResetCalls != 0 {
		t.Error("Execute() should not send an email for an inactive account")
	}
}

func TestRequestPasswordResetRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	requestReset := newSyncRequestPasswordReset(&gatewayfake.UserRepository{}, &gatewayfake.Mailer{}, NewPasswordResetTokens(), testResetURL)

	err := requestReset.Execute(context.Background(), "not-an-email")

	if err != domain.ErrEmailInvalido {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailInvalido)
	}
}

func TestRequestPasswordResetRejectsWithinCooldown(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByEmail: registeredActiveUser()}
	mailer := &gatewayfake.Mailer{}
	requestReset := newSyncRequestPasswordReset(repository, mailer, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()})

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
	requestReset := newSyncRequestPasswordReset(repository, &gatewayfake.Mailer{}, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()})

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
	useCase := newSyncRequestPasswordReset(repository, &gatewayfake.Mailer{}, NewPasswordResetTokens(), testResetURL, clock)

	for i := 0; i < 5; i++ {
		email := fmt.Sprintf("made-up-%d@example.com", i)
		if err := useCase.Execute(context.Background(), email); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	}
	if len(useCase.lastSent) != 5 {
		t.Fatalf("len(lastSent) = %d, want 5 before the cooldown elapses", len(useCase.lastSent))
	}

	clock.now = clock.now.Add(passwordResetRequestCooldown)
	if err := useCase.Execute(context.Background(), "yet-another@example.com"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(useCase.lastSent) != 1 {
		t.Errorf("len(lastSent) = %d, want 1 — the sweep should have dropped the 5 stale entries", len(useCase.lastSent))
	}
}

// TestRequestPasswordResetBoundsCooldownEntries guards against unbounded
// memory growth within a single cooldown window: even before any entry is
// old enough to sweep, the map must not grow past a hard cap no matter how
// many distinct made-up emails an attacker sends.
func TestRequestPasswordResetBoundsCooldownEntries(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByEmailErr: domain.ErrUsuarioNoEncontrado}
	clock := &mutableClock{now: time.Now()}
	useCase := newSyncRequestPasswordReset(repository, &gatewayfake.Mailer{}, NewPasswordResetTokens(), testResetURL, clock)

	for i := 0; i < maxTrackedResetEmails+10; i++ {
		email := fmt.Sprintf("made-up-%d@example.com", i)
		if err := useCase.Execute(context.Background(), email); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	}

	if len(useCase.lastSent) > maxTrackedResetEmails {
		t.Errorf("len(lastSent) = %d, want at most %d", len(useCase.lastSent), maxTrackedResetEmails)
	}
}

func TestRequestPasswordResetAllowsAfterCooldownElapses(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByEmail: registeredActiveUser()}
	mailer := &gatewayfake.Mailer{}
	clock := &mutableClock{now: time.Now()}
	requestReset := newSyncRequestPasswordReset(repository, mailer, NewPasswordResetTokens(), testResetURL, clock)

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

func TestRequestPasswordResetCooldownIgnoresEmailCasing(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByEmail: registeredActiveUser()}
	mailer := &gatewayfake.Mailer{}
	requestReset := newSyncRequestPasswordReset(repository, mailer, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()})

	if err := requestReset.Execute(context.Background(), "ana@example.com"); err != nil {
		t.Fatalf("Execute() first call error = %v", err)
	}

	for _, variant := range []string{"ANA@example.com", "Ana@Example.COM", "  ana@EXAMPLE.com  "} {
		if err := requestReset.Execute(context.Background(), variant); !errors.Is(err, domain.ErrPasswordResetRateLimited) {
			t.Errorf("Execute(%q) error = %v, want %v", variant, err, domain.ErrPasswordResetRateLimited)
		}
	}
	if mailer.SendPasswordResetCalls != 1 {
		t.Errorf("Execute() mailer.SendPasswordReset calls = %d, want 1 across every casing of the same email", mailer.SendPasswordResetCalls)
	}
}

func TestRequestPasswordResetLooksUpTheNormalizedEmail(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByEmail: registeredActiveUser()}
	requestReset := newSyncRequestPasswordReset(repository, &gatewayfake.Mailer{}, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()})

	if err := requestReset.Execute(context.Background(), "  ANA@Example.com "); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if repository.FindByEmailInput != "ana@example.com" {
		t.Errorf("Execute() looked up %q, want %q", repository.FindByEmailInput, "ana@example.com")
	}
}

func TestRequestPasswordResetCapsBackgroundJobsInFlight(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByEmailErr: domain.ErrUsuarioNoEncontrado}
	requestReset := NewRequestPasswordReset(repository, &gatewayfake.Mailer{}, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()}).(*requestPasswordResetUseCase)
	var pending []func()
	requestReset.dispatch = func(work func()) { pending = append(pending, work) }

	if err := requestReset.Execute(context.Background(), "user-0@example.com"); err != nil {
		t.Fatalf("Execute() first call error = %v", err)
	}
	if err := requestReset.Execute(context.Background(), "user-0@example.com"); !errors.Is(err, domain.ErrPasswordResetRateLimited) {
		t.Fatalf("Execute() repeated call error = %v, want %v", err, domain.ErrPasswordResetRateLimited)
	}
	for index := 1; index < maxConcurrentPasswordResets; index++ {
		if err := requestReset.Execute(context.Background(), fmt.Sprintf("user-%d@example.com", index)); err != nil {
			t.Fatalf("Execute() call %d error = %v, want a free slot: a cooldown rejection must not keep one", index, err)
		}
	}

	err := requestReset.Execute(context.Background(), "one-too-many@example.com")
	if !errors.Is(err, domain.ErrPasswordResetBusy) {
		t.Fatalf("Execute() over the cap error = %v, want %v", err, domain.ErrPasswordResetBusy)
	}
	if len(pending) != maxConcurrentPasswordResets {
		t.Errorf("Execute() dispatched %d jobs, want at most %d", len(pending), maxConcurrentPasswordResets)
	}

	pending[0]()

	if err := requestReset.Execute(context.Background(), "one-too-many@example.com"); err != nil {
		t.Errorf("Execute() after a job finished error = %v, want the rejected email to go through: being busy must not start its cooldown", err)
	}
}

type lookupContextRepository struct {
	*gatewayfake.UserRepository
	onLookup func(context.Context)
}

func (repository *lookupContextRepository) FindByEmail(ctx context.Context, email string) (domain.Usuario, error) {
	repository.onLookup(ctx)
	return repository.UserRepository.FindByEmail(ctx, email)
}

func TestRequestPasswordResetJobSurvivesRequestCancellationWithItsOwnDeadline(t *testing.T) {
	t.Parallel()

	var lookupErr error
	var hasDeadline bool
	repository := &lookupContextRepository{
		UserRepository: &gatewayfake.UserRepository{FindByEmailErr: domain.ErrUsuarioNoEncontrado},
		onLookup: func(ctx context.Context) {
			lookupErr = ctx.Err()
			_, hasDeadline = ctx.Deadline()
		},
	}
	requestReset := newSyncRequestPasswordReset(repository, &gatewayfake.Mailer{}, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()})
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	cancelRequest()

	if err := requestReset.Execute(requestCtx, "ana@example.com"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if lookupErr != nil {
		t.Errorf("lookup context error = %v, want the job to outlive the cancelled request", lookupErr)
	}
	if !hasDeadline {
		t.Error("lookup context has no deadline, want the job bounded by its own timeout")
	}
}

func TestRequestPasswordResetReleasesTheSlotOfAHungJob(t *testing.T) {
	t.Parallel()

	repository := &lookupContextRepository{
		UserRepository: &gatewayfake.UserRepository{FindByEmailErr: domain.ErrUsuarioNoEncontrado},
		onLookup:       func(ctx context.Context) { <-ctx.Done() },
	}
	requestReset := NewRequestPasswordReset(repository, &gatewayfake.Mailer{}, NewPasswordResetTokens(), testResetURL, fixedClock{now: time.Now()}).(*requestPasswordResetUseCase)
	requestReset.workTimeout = 20 * time.Millisecond

	for index := 0; index < maxConcurrentPasswordResets; index++ {
		if err := requestReset.Execute(context.Background(), fmt.Sprintf("user-%d@example.com", index)); err != nil {
			t.Fatalf("Execute() call %d error = %v", index, err)
		}
	}
	if err := requestReset.Execute(context.Background(), "waiting@example.com"); !errors.Is(err, domain.ErrPasswordResetBusy) {
		t.Fatalf("Execute() with every slot hung error = %v, want %v", err, domain.ErrPasswordResetBusy)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		err := requestReset.Execute(context.Background(), "waiting@example.com")
		if err == nil {
			return
		}
		if !errors.Is(err, domain.ErrPasswordResetBusy) || time.Now().After(deadline) {
			t.Fatalf("Execute() after the hung jobs timed out error = %v, want a released slot", err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
