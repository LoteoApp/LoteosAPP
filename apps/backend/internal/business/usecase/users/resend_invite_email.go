package users

import (
	"context"
	"sync"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// ResendInviteEmail mints a fresh invite link and sends the invite email
// again, for when CreateUser's own best-effort send failed. The original
// link is never persisted (it only ever lives in memory during one
// request), so a resend can't reuse it — it has to mint a new one.
type ResendInviteEmail interface {
	Execute(ctx context.Context, actorRoles []string, id string) error
}

const resendInviteCooldown = 60 * time.Second

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

type resendInviteEmailUseCase struct {
	repository gateway.UserRepository
	identity   gateway.IdentityProvider
	mailer     gateway.Mailer
	clock      Clock

	mu       sync.Mutex
	lastSent map[string]time.Time
}

func NewResendInviteEmail(repository gateway.UserRepository, identity gateway.IdentityProvider, mailer gateway.Mailer, clocks ...Clock) ResendInviteEmail {
	clock := Clock(SystemClock{})
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &resendInviteEmailUseCase{
		repository: repository,
		identity:   identity,
		mailer:     mailer,
		clock:      clock,
		lastSent:   make(map[string]time.Time),
	}
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
	if !useCase.reserve(target.ID) {
		return domain.ErrInviteEmailRateLimited
	}

	inviteURL, err := useCase.identity.GenerateInviteLink(ctx, target.Email)
	if err != nil {
		return fromRepository(err)
	}

	if err := useCase.mailer.SendUserInvite(ctx, gateway.UserInviteEmail{
		To:        target.Email,
		Nombre:    target.Nombre,
		Apellido:  target.Apellido,
		Rol:       target.Rol,
		InviteURL: inviteURL,
	}); err != nil {
		return domain.ErrInviteEmailUnavailable.WithCause(err)
	}

	return nil
}

func (useCase *resendInviteEmailUseCase) reserve(usuarioID string) bool {
	useCase.mu.Lock()
	defer useCase.mu.Unlock()

	now := useCase.clock.Now()
	if last, ok := useCase.lastSent[usuarioID]; ok && now.Sub(last) < resendInviteCooldown {
		return false
	}
	useCase.lastSent[usuarioID] = now
	return true
}
