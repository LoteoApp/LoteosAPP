package users

import (
	"context"
	"errors"
	"log/slog"
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
		lastSent:   make(map[string]time.Time),
	}
}

func (useCase *requestPasswordResetUseCase) Execute(ctx context.Context, email string) error {
	email = strings.TrimSpace(email)
	if !domain.EmailValido(email) {
		return domain.ErrEmailInvalido
	}
	if !useCase.reserve(email) {
		return domain.ErrPasswordResetRateLimited
	}

	usuario, err := useCase.repository.FindByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
			slog.ErrorContext(ctx, "password reset lookup failed", "error", err)
		}
		return nil
	}
	if !usuario.Activo() {
		return nil
	}

	token, err := useCase.tokens.Issue(usuario.ID, useCase.clock.Now(), PasswordResetTokenTTL)
	if err != nil {
		slog.ErrorContext(ctx, "password reset token generation failed", "error", err)
		return nil
	}

	if err := useCase.mailer.SendPasswordReset(ctx, gateway.PasswordResetEmail{
		To:       usuario.Email,
		Nombre:   usuario.Nombre,
		Apellido: usuario.Apellido,
		ResetURL: useCase.resetURL + "?token=" + token,
	}); err != nil {
		slog.ErrorContext(ctx, "password reset email send failed", "usuario_id", usuario.ID, "error", err)
	}

	return nil
}

func (useCase *requestPasswordResetUseCase) reserve(email string) bool {
	useCase.mu.Lock()
	defer useCase.mu.Unlock()

	now := useCase.clock.Now()
	if last, ok := useCase.lastSent[email]; ok && now.Sub(last) < passwordResetRequestCooldown {
		return false
	}
	useCase.lastSent[email] = now
	return true
}
