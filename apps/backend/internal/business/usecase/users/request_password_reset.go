package users

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// passwordResetRequestCooldown mirrors resendInviteCooldown's reasoning, but
// is checked before the email is even looked up: rate-limiting on the raw
// input, not on whether it matched an account, keeps the cooldown itself
// from becoming a way to tell registered emails apart from made-up ones.
const passwordResetRequestCooldown = 60 * time.Second

// maxConcurrentPasswordResets caps how many lookup-and-send jobs can be in
// flight at once. The endpoint is unauthenticated and each accepted request
// spawns background work, so without a cap a burst of distinct emails would
// spawn goroutines and database lookups without limit.
const maxConcurrentPasswordResets = 16

// maxTrackedResetEmails caps lastSent so a burst of distinct made-up emails
// within a single cooldown window can't grow it without bound — the sweep
// alone only reclaims entries once they age out.
const maxTrackedResetEmails = 10_000

// RequestPasswordReset sends a password reset link to email if it belongs to
// an active account. It always succeeds from the caller's point of view
// (except for a malformed email or a cooldown hit, neither of which reveals
// whether the account exists) — a real send failure is logged, not
// propagated, so the response can't be used to probe which emails are
// registered.
type RequestPasswordReset interface {
	Execute(ctx context.Context, email string) error
}

type requestPasswordResetUseCase struct {
	repository gateway.UserRepository
	mailer     gateway.Mailer
	tokens     *PasswordResetTokens
	resetURL   string
	clock      Clock
	// dispatch runs the lookup-and-send work. Defaults to a real goroutine;
	// tests overwrite it to run inline so assertions after Execute returns
	// see its effects deterministically.
	dispatch func(func())
	inFlight chan struct{}

	mu       sync.Mutex
	lastSent map[string]time.Time
}

func NewRequestPasswordReset(
	repository gateway.UserRepository,
	mailer gateway.Mailer,
	tokens *PasswordResetTokens,
	resetURL string,
	clocks ...Clock,
) RequestPasswordReset {
	clock := Clock(SystemClock{})
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &requestPasswordResetUseCase{
		repository: repository,
		mailer:     mailer,
		tokens:     tokens,
		resetURL:   resetURL,
		clock:      clock,
		dispatch:   func(work func()) { go work() },
		inFlight:   make(chan struct{}, maxConcurrentPasswordResets),
		lastSent:   make(map[string]time.Time),
	}
}

// Execute never waits on the lookup or the send: for an existing, active
// account those cost a token generation and a call to the mail provider,
// while a made-up or inactive email returns right after the lookup. Awaiting
// that work here would make the response measurably slower for a real
// account than for a fake one, letting a caller enumerate accounts by
// timing alone even though both eventually reply 204.
func (useCase *requestPasswordResetUseCase) Execute(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if !domain.EmailValido(email) {
		return domain.ErrEmailInvalido
	}

	select {
	case useCase.inFlight <- struct{}{}:
	default:
		return domain.ErrPasswordResetBusy
	}
	if !useCase.reserve(email) {
		<-useCase.inFlight
		return domain.ErrPasswordResetRateLimited
	}

	detached := context.WithoutCancel(ctx)
	useCase.dispatch(func() {
		defer func() { <-useCase.inFlight }()
		useCase.sendResetLink(detached, email)
	})

	return nil
}

func (useCase *requestPasswordResetUseCase) sendResetLink(ctx context.Context, email string) {
	usuario, err := useCase.repository.FindByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			slog.ErrorContext(ctx, "password reset lookup failed", "error", err)
		}
		return
	}
	if !usuario.Activo() {
		return
	}

	token, err := useCase.tokens.Issue(usuario.ID, useCase.clock.Now(), PasswordResetTokenTTL)
	if err != nil {
		slog.ErrorContext(ctx, "password reset token generation failed", "error", err)
		return
	}

	// The token travels in the URL fragment, not the query string: a
	// fragment never leaves the browser, so it's absent from proxy/nginx
	// access logs and from browser history synced elsewhere.
	if err := useCase.mailer.SendPasswordReset(ctx, gateway.PasswordResetEmail{
		To:       usuario.Email,
		Nombre:   usuario.Nombre,
		Apellido: usuario.Apellido,
		ResetURL: useCase.resetURL + "#token=" + url.QueryEscape(token),
	}); err != nil {
		slog.ErrorContext(ctx, "password reset email send failed", "usuario_id", usuario.ID, "error", err)
	}
}

func (useCase *requestPasswordResetUseCase) reserve(email string) bool {
	useCase.mu.Lock()
	defer useCase.mu.Unlock()

	now := useCase.clock.Now()
	useCase.sweepLocked(now)
	if last, ok := useCase.lastSent[email]; ok && now.Sub(last) < passwordResetRequestCooldown {
		return false
	}
	if len(useCase.lastSent) >= maxTrackedResetEmails {
		useCase.evictOneLocked()
	}
	useCase.lastSent[email] = now
	return true
}

// evictOneLocked drops a single entry so a new one can be tracked once
// lastSent is at capacity. Go's map iteration order is randomized per run,
// so this evicts an effectively arbitrary entry rather than the oldest one
// — acceptable here since the cap only exists to bound memory, not to keep
// the cooldown precise under sustained abuse. Caller must hold mu.
func (useCase *requestPasswordResetUseCase) evictOneLocked() {
	for email := range useCase.lastSent {
		delete(useCase.lastSent, email)
		return
	}
}

// sweepLocked drops cooldown entries older than the window, so lastSent
// stays bounded to recent attempts instead of growing with every distinct
// email ever tried since the process started — this endpoint is
// unauthenticated, so that input isn't limited to real accounts. Caller must
// hold mu.
func (useCase *requestPasswordResetUseCase) sweepLocked(now time.Time) {
	for email, last := range useCase.lastSent {
		if now.Sub(last) >= passwordResetRequestCooldown {
			delete(useCase.lastSent, email)
		}
	}
}
