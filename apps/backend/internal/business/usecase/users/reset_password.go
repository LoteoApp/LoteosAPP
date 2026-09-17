package users

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

const minPasswordLength = 8

// ResetPassword confirms a password reset requested through
// RequestPasswordReset: it redeems the one-time token and, only if that
// succeeds, sets the caller's chosen password. The token is consumed
// (invalidated) as soon as it's read, whether or not the rest of the call
// succeeds, so it can never be replayed.
type ResetPassword interface {
	Execute(ctx context.Context, token, newPassword string) error
}

type resetPasswordUseCase struct {
	repository gateway.UserRepository
	identity   gateway.IdentityProvider
	tokens     *PasswordResetTokens
	clock      Clock
}

func NewResetPassword(
	repository gateway.UserRepository,
	identity gateway.IdentityProvider,
	tokens *PasswordResetTokens,
	clocks ...Clock,
) ResetPassword {
	clock := Clock(SystemClock{})
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &resetPasswordUseCase{repository: repository, identity: identity, tokens: tokens, clock: clock}
}

func (useCase *resetPasswordUseCase) Execute(ctx context.Context, token, newPassword string) error {
	if len(newPassword) < minPasswordLength {
		return domain.ErrPasswordInvalido
	}

	usuarioID, ok := useCase.tokens.Consume(token, useCase.clock.Now())
	if !ok {
		return domain.ErrPasswordResetTokenInvalido
	}

	usuario, err := useCase.repository.FindByID(ctx, usuarioID)
	if err != nil {
		return fromRepository(err)
	}
	if !usuario.Activo() {
		return domain.ErrUsuarioDadoDeBaja
	}

	if err := useCase.identity.SetPassword(ctx, usuario.AuthProviderID, newPassword); err != nil {
		return fromRepository(err)
	}

	return nil
}
