package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func activeManagedUserWithContact() domain.Usuario {
	usuario := activeManagedUser()
	usuario.AuthProviderID = "sb-123"
	usuario.Email = "ana@example.com"
	return usuario
}

func TestResendInviteEmailRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: activeManagedUserWithContact()}
	identity := &gatewayfake.IdentityProvider{}
	mailer := &gatewayfake.Mailer{}
	resendInviteEmail := NewResendInviteEmail(repository, identity, mailer)

	err := resendInviteEmail.Execute(context.Background(), []string{domain.RolAdministrativo}, "user-1")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if identity.GenerateInviteLinkCalls != 0 {
		t.Error("Execute() should not generate a link when actor is not administrador")
	}
}

func TestResendInviteEmailHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: activeManagedUserWithContact()}
	identity := &gatewayfake.IdentityProvider{GenerateInviteLinkURL: "https://app.loteosapp.com/aceptar-invitacion?token_hash=fresh"}
	mailer := &gatewayfake.Mailer{}
	resendInviteEmail := NewResendInviteEmail(repository, identity, mailer)

	if err := resendInviteEmail.Execute(context.Background(), []string{domain.RolAdministrador}, "user-1"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if identity.GenerateInviteLinkCalls != 1 || identity.GenerateInviteLinkEmail != "ana@example.com" {
		t.Errorf("Execute() identity.GenerateInviteLink calls = %d, email = %q, want 1 call for ana@example.com",
			identity.GenerateInviteLinkCalls, identity.GenerateInviteLinkEmail)
	}
	if mailer.SendUserInviteCalls != 1 {
		t.Fatalf("Execute() mailer.SendUserInvite calls = %d, want 1", mailer.SendUserInviteCalls)
	}
	sent := mailer.SendUserInviteInputs[0]
	if sent.To != "ana@example.com" || sent.InviteURL != identity.GenerateInviteLinkURL {
		t.Errorf("Execute() sent invite = %#v", sent)
	}
}

func TestResendInviteEmailReportsAnAlreadyAcceptedInvite(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: activeManagedUserWithContact()}
	identity := &gatewayfake.IdentityProvider{GenerateInviteLinkErr: domain.ErrEmailEnUso}
	mailer := &gatewayfake.Mailer{}
	resendInviteEmail := NewResendInviteEmail(repository, identity, mailer)

	err := resendInviteEmail.Execute(context.Background(), []string{domain.RolAdministrador}, "user-1")

	if !errors.Is(err, domain.ErrInviteAlreadyAccepted) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInviteAlreadyAccepted)
	}
	if mailer.SendUserInviteCalls != 0 {
		t.Errorf("Execute() mailer.SendUserInvite calls = %d, want 0 when there is no link to send", mailer.SendUserInviteCalls)
	}
}

func TestResendInviteEmailRejectsUnknownID(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FindByIDErr: domain.ErrUsuarioNoEncontrado}
	resendInviteEmail := NewResendInviteEmail(repository, &gatewayfake.IdentityProvider{}, &gatewayfake.Mailer{})

	err := resendInviteEmail.Execute(context.Background(), []string{domain.RolAdministrador}, "user-1")

	if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
	}
}

func TestResendInviteEmailRejectsRolesThisABMDoesNotManage(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: domain.Usuario{ID: "user-2", Rol: domain.RolAdministrador}}
	resendInviteEmail := NewResendInviteEmail(repository, &gatewayfake.IdentityProvider{}, &gatewayfake.Mailer{})

	err := resendInviteEmail.Execute(context.Background(), []string{domain.RolAdministrador}, "user-2")

	if !errors.Is(err, domain.ErrUsuarioNoEncontrado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioNoEncontrado)
	}
}

func TestResendInviteEmailRejectsInactiveUser(t *testing.T) {
	t.Parallel()

	baja := time.Now()
	inactive := activeManagedUserWithContact()
	inactive.FechaBaja = &baja
	repository := &gatewayfake.UserRepository{FoundByID: inactive}
	resendInviteEmail := NewResendInviteEmail(repository, &gatewayfake.IdentityProvider{}, &gatewayfake.Mailer{})

	err := resendInviteEmail.Execute(context.Background(), []string{domain.RolAdministrador}, "user-1")

	if !errors.Is(err, domain.ErrUsuarioDadoDeBaja) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrUsuarioDadoDeBaja)
	}
}

func TestResendInviteEmailPropagatesGenerateInviteLinkFailure(t *testing.T) {
	t.Parallel()

	cause := errors.New("supabase unavailable")
	repository := &gatewayfake.UserRepository{FoundByID: activeManagedUserWithContact()}
	identity := &gatewayfake.IdentityProvider{GenerateInviteLinkErr: cause}
	mailer := &gatewayfake.Mailer{}
	resendInviteEmail := NewResendInviteEmail(repository, identity, mailer)

	err := resendInviteEmail.Execute(context.Background(), []string{domain.RolAdministrador}, "user-1")

	assertDatabaseUnavailable(t, err, cause)
	if mailer.SendUserInviteCalls != 0 {
		t.Error("Execute() should not attempt to send when generating the link already failed")
	}
}

func TestResendInviteEmailSurfacesSendFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{FoundByID: activeManagedUserWithContact()}
	identity := &gatewayfake.IdentityProvider{GenerateInviteLinkURL: "https://app.loteosapp.com/aceptar-invitacion?token_hash=fresh"}
	mailer := &gatewayfake.Mailer{SendUserInviteErr: errors.New("resend unavailable")}
	resendInviteEmail := NewResendInviteEmail(repository, identity, mailer)

	err := resendInviteEmail.Execute(context.Background(), []string{domain.RolAdministrador}, "user-1")

	if !errors.Is(err, domain.ErrInviteEmailUnavailable) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInviteEmailUnavailable)
	}
}
