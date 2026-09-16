package users

import (
	"context"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ResendInviteEmail mints a fresh temporary password and sends the invite
// email again, for when CreateUser's own best-effort send failed. The
// original temporary password is never persisted (it only ever lives in
// memory during one request), so a resend can't reuse it — it has to reset
// the account's password before mailing a new one.
type ResendInviteEmail interface {
	Execute(ctx context.Context, actorRoles []string, id string) error
}

type resendInviteEmailUseCase struct {
	repository gateway.UserRepository
	identity   gateway.IdentityProvider
	mailer     gateway.Mailer
	loginURL   string
}

func NewResendInviteEmail(repository gateway.UserRepository, identity gateway.IdentityProvider, mailer gateway.Mailer, loginURL string) ResendInviteEmail {
	return &resendInviteEmailUseCase{repository: repository, identity: identity, mailer: mailer, loginURL: loginURL}
}

func (useCase *resendInviteEmailUseCase) Execute(ctx context.Context, actorRoles []string, id string) error {
	if !domain.HasRole(actorRoles, domain.RolAdministrador) {
		return domain.ErrNoAutorizado
	}

	target, err := useCase.repository.FindByID(ctx, id)
	if err != nil {
		return fromRepository(err)
	}
	if !esRolGestionable(target.Rol) {
		return domain.ErrUsuarioNoEncontrado
	}
	if !target.Activo() {
		return domain.ErrUsuarioDadoDeBaja
	}

	temporaryPassword, err := useCase.identity.ResetTemporaryPassword(ctx, target.AuthProviderID)
	if err != nil {
		return fromRepository(err)
	}

	if err := useCase.mailer.SendUserInvite(ctx, gateway.UserInviteEmail{
		To:                target.Email,
		Nombre:            target.Nombre,
		Apellido:          target.Apellido,
		Rol:               target.Rol,
		TemporaryPassword: temporaryPassword,
		LoginURL:          useCase.loginURL,
	}); err != nil {
		return domain.ErrInviteEmailUnavailable.WithCause(err)
	}

	return nil
}
