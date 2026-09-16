package users

import (
	"context"
	"log/slog"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// CreateUser gives a new user access to the system. Only callers with the
// administrador role may do this. The role must be one of
// gestionableRoles — this ABM doesn't create other administrador accounts.
// agencyID is required when rol is inmobiliaria (the agency the user
// operates on behalf of) and ignored for every other role.
type CreateUser interface {
	// The bool result reports whether the invite email went out; a false
	// with a nil error means the usuario was created but the caller should
	// fall back to communicating the temporary password some other way (or
	// retry via ResendInviteEmail) — it never fails the request.
	Execute(ctx context.Context, actorRoles []string, nombre, apellido, email, rol, agencyID string) (domain.Usuario, string, bool, error)
}

type createUserUseCase struct {
	repository gateway.UserRepository
	identity   gateway.IdentityProvider
	agencies   gateway.AgencyRepository
	mailer     gateway.Mailer
	loginURL   string
}

func NewCreateUser(
	repository gateway.UserRepository,
	identity gateway.IdentityProvider,
	agencies gateway.AgencyRepository,
	mailer gateway.Mailer,
	loginURL string,
) CreateUser {
	return &createUserUseCase{repository: repository, identity: identity, agencies: agencies, mailer: mailer, loginURL: loginURL}
}

// Execute creates the account in the identity provider first and, if
// persisting the local profile then fails, removes it again so no orphaned
// account is left behind.
func (useCase *createUserUseCase) Execute(
	ctx context.Context,
	actorRoles []string,
	nombre, apellido, email, rol, agencyID string,
) (domain.Usuario, string, bool, error) {
	if !domain.HasRole(actorRoles, domain.RolAdministrador) {
		return domain.Usuario{}, "", false, domain.ErrNoAutorizado
	}

	nombre = strings.TrimSpace(nombre)
	apellido = strings.TrimSpace(apellido)
	if nombre == "" || apellido == "" {
		return domain.Usuario{}, "", false, domain.ErrPerfilInvalido
	}

	email = strings.TrimSpace(email)
	if !domain.EmailValido(email) {
		return domain.Usuario{}, "", false, domain.ErrEmailInvalido
	}

	if !esRolGestionable(domain.Rol(rol)) {
		return domain.Usuario{}, "", false, domain.ErrRolInvalido
	}

	// Only rol inmobiliaria carries an agency: any id sent for another role
	// is ignored rather than persisted or validated.
	var agencyIDPtr *string
	if domain.Rol(rol) == domain.RolInmobiliaria {
		agencyID = strings.TrimSpace(agencyID)
		if agencyID == "" {
			return domain.Usuario{}, "", false, domain.ErrAgenciaRequerida
		}
		if _, err := useCase.agencies.FindByID(ctx, agencyID); err != nil {
			return domain.Usuario{}, "", false, fromRepository(err)
		}
		agencyIDPtr = &agencyID
	}

	authProviderID, temporaryPassword, err := useCase.identity.CreateUser(ctx, email, rol)
	if err != nil {
		return domain.Usuario{}, "", false, fromRepository(err)
	}

	usuario, err := useCase.repository.Create(ctx, domain.Usuario{
		AuthProviderID: authProviderID,
		Email:          email,
		Nombre:         nombre,
		Apellido:       apellido,
		Rol:            domain.Rol(rol),
		AgencyID:       agencyIDPtr,
		PerfilCompleto: true,
	})
	if err != nil {
		if deleteErr := useCase.identity.DeleteUser(ctx, authProviderID); deleteErr != nil {
			slog.ErrorContext(ctx, "compensating identity provider delete failed after local persistence error",
				"auth_provider_id", authProviderID, "error", deleteErr)
		}
		return domain.Usuario{}, "", false, fromRepository(err)
	}

	inviteEmailSent := useCase.sendInvite(ctx, usuario, temporaryPassword)

	return usuario, temporaryPassword, inviteEmailSent, nil
}

// sendInvite is a best-effort side effect: the usuario already exists (and
// the admin already has temporaryPassword to hand over some other way), so a
// send failure is only logged, never returned as an error. The caller can
// retry it on demand via ResendInviteEmail.
func (useCase *createUserUseCase) sendInvite(ctx context.Context, usuario domain.Usuario, temporaryPassword string) bool {
	err := useCase.mailer.SendUserInvite(ctx, gateway.UserInviteEmail{
		To:                usuario.Email,
		Nombre:            usuario.Nombre,
		Apellido:          usuario.Apellido,
		Rol:               usuario.Rol,
		TemporaryPassword: temporaryPassword,
		LoginURL:          useCase.loginURL,
	})
	if err != nil {
		slog.ErrorContext(ctx, "invite email send failed", "usuario_id", usuario.ID, "error", err)
		return false
	}
	return true
}
