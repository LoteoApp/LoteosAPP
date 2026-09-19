package users

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestCreateUserRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{}
	createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, &gatewayfake.Mailer{})

	_, _, err := createUser.Execute(context.Background(), []string{"administrativo"}, "Ana", "Gómez", "ana@example.com", domain.RolAdministrativo, "")

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if identity.CreateCalls != 0 {
		t.Error("Execute() should not call identity provider when actor is not administrador")
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not call repository when actor is not administrador")
	}
}

func TestCreateUserRejectsIncompleteProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nombre   string
		apellido string
	}{
		{name: "sin nombre", apellido: "Gómez"},
		{name: "sin apellido", nombre: "Ana"},
		{name: "solo espacios", nombre: "   ", apellido: "   "},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.UserRepository{}
			identity := &gatewayfake.IdentityProvider{}
			createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, &gatewayfake.Mailer{})

			_, _, err := createUser.Execute(context.Background(),
				[]string{domain.RolAdministrador}, test.nombre, test.apellido, "ana@example.com", domain.RolAdministrativo, "")

			if !errors.Is(err, domain.ErrPerfilInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrPerfilInvalido)
			}
			if identity.CreateCalls != 0 {
				t.Error("Execute() should not call the identity provider with an incomplete profile")
			}
		})
	}
}

func TestCreateUserRejectsInvalidRol(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{}
	createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, &gatewayfake.Mailer{})

	_, _, err := createUser.Execute(context.Background(), []string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", "superadmin", "")

	if !errors.Is(err, domain.ErrRolInvalido) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrRolInvalido)
	}
	if identity.CreateCalls != 0 {
		t.Error("Execute() should not call identity provider when rol is invalid")
	}
}

func TestCreateUserRejectsRolesThisABMDoesNotManage(t *testing.T) {
	t.Parallel()

	// administrador isn't created through this route: it's a valid
	// domain.Rol value, but not one this use case accepts.
	for _, rol := range []string{domain.RolAdministrador} {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			repository := &gatewayfake.UserRepository{}
			identity := &gatewayfake.IdentityProvider{}
			createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, &gatewayfake.Mailer{})

			_, _, err := createUser.Execute(context.Background(), []string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", rol, "")

			if !errors.Is(err, domain.ErrRolInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrRolInvalido)
			}
			if identity.CreateCalls != 0 {
				t.Error("Execute() should not call identity provider for a role this ABM doesn't manage")
			}
		})
	}
}

func TestCreateUserRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{}
	createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, &gatewayfake.Mailer{})

	_, _, err := createUser.Execute(context.Background(), []string{domain.RolAdministrador}, "Ana", "Gómez", "not-an-email", domain.RolAdministrativo, "")

	if !errors.Is(err, domain.ErrEmailInvalido) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailInvalido)
	}
	if identity.CreateCalls != 0 {
		t.Error("Execute() should not call identity provider when email is invalid")
	}
}

func TestCreateUserHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", InviteURL: "https://app.loteosapp.com/aceptar-invitacion?token_hash=abc"}
	createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, &gatewayfake.Mailer{})

	usuario, inviteEmailSent, err := createUser.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "  Ana  ", "  Gómez  ", "  ana@example.com  ", domain.RolAdministrativo, "")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if usuario.AuthProviderID != "sb-123" {
		t.Errorf("Execute() auth provider id = %q, want %q", usuario.AuthProviderID, "sb-123")
	}
	if usuario.Email != "ana@example.com" {
		t.Errorf("Execute() email = %q, want %q", usuario.Email, "ana@example.com")
	}
	if usuario.Nombre != "Ana" || usuario.Apellido != "Gómez" {
		t.Errorf("Execute() should trim the profile, got %q %q", usuario.Nombre, usuario.Apellido)
	}
	if !usuario.PerfilCompleto {
		t.Error("Execute() should mark the profile as complete when nombre and apellido are given")
	}
	if !usuario.Activo() {
		t.Error("Execute() should create an active user")
	}
	if !inviteEmailSent {
		t.Error("Execute() should report the invite email as sent when the mailer succeeds")
	}
	if usuario.InvitacionAceptada == nil || *usuario.InvitacionAceptada {
		t.Errorf("Execute() InvitacionAceptada = %v, want a pending invitation", usuario.InvitacionAceptada)
	}
	if repository.CreateCalls != 1 {
		t.Errorf("Execute() repository.Create calls = %d, want 1", repository.CreateCalls)
	}
	if identity.DeleteCalls != 0 {
		t.Error("Execute() should not compensate when persistence succeeds")
	}
}

func TestCreateUserPropagatesIdentityProviderError(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{CreateErr: domain.ErrEmailEnUso}
	createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, &gatewayfake.Mailer{})

	_, _, err := createUser.Execute(context.Background(), []string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", domain.RolAdministrativo, "")

	if !errors.Is(err, domain.ErrEmailEnUso) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailEnUso)
	}
	if repository.CreateCalls != 0 {
		t.Error("Execute() should not call repository when identity provider fails")
	}
}

func TestCreateUserCompensatesWhenPersistenceFails(t *testing.T) {
	t.Parallel()

	persistErr := errors.New("insert failed")
	repository := &gatewayfake.UserRepository{CreateErr: persistErr}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", InviteURL: "https://app.loteosapp.com/aceptar-invitacion?token_hash=abc"}
	createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, &gatewayfake.Mailer{})

	_, _, err := createUser.Execute(context.Background(), []string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", domain.RolAdministrativo, "")

	assertDatabaseUnavailable(t, err, persistErr)
	if identity.DeleteCalls != 1 {
		t.Fatalf("Execute() identity.DeleteUser calls = %d, want 1", identity.DeleteCalls)
	}
	if identity.DeletedUserID != "sb-123" {
		t.Errorf("Execute() compensated user id = %q, want %q", identity.DeletedUserID, "sb-123")
	}
}

func TestCreateUserReturnsOriginalErrorWhenCompensationAlsoFails(t *testing.T) {
	t.Parallel()

	persistErr := errors.New("insert failed")
	repository := &gatewayfake.UserRepository{CreateErr: persistErr}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", DeleteErr: errors.New("delete failed")}
	createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, &gatewayfake.Mailer{})

	_, _, err := createUser.Execute(context.Background(), []string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", domain.RolAdministrativo, "")

	assertDatabaseUnavailable(t, err, persistErr)
}

func TestCreateUserSendsInviteEmailOnSuccess(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", InviteURL: "https://app.loteosapp.com/aceptar-invitacion?token_hash=abc"}
	mailer := &gatewayfake.Mailer{}
	createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, mailer)

	_, inviteEmailSent, err := createUser.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", domain.RolAdministrativo, "")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !inviteEmailSent {
		t.Error("Execute() inviteEmailSent = false, want true")
	}
	if mailer.SendUserInviteCalls != 1 {
		t.Fatalf("Execute() mailer.SendUserInvite calls = %d, want 1", mailer.SendUserInviteCalls)
	}
	sent := mailer.SendUserInviteInputs[0]
	if sent.To != "ana@example.com" || sent.Nombre != "Ana" || sent.Apellido != "Gómez" ||
		sent.Rol != domain.RolAdministrativo || sent.InviteURL != identity.InviteURL {
		t.Errorf("Execute() sent invite = %#v", sent)
	}
}

// TestCreateUserSurvivesInviteEmailFailure is the behavior a mail provider
// outage must never turn into a failed user creation: the usuario already
// exists with no password, and the admin can trigger ResendInviteEmail once
// the mail provider is back.
func TestCreateUserSurvivesInviteEmailFailure(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", InviteURL: "https://app.loteosapp.com/aceptar-invitacion?token_hash=abc"}
	mailer := &gatewayfake.Mailer{SendUserInviteErr: errors.New("resend unavailable")}
	createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, mailer)

	usuario, inviteEmailSent, err := createUser.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", domain.RolAdministrativo, "")
	if err != nil {
		t.Fatalf("Execute() error = %v, want the send failure to be swallowed", err)
	}
	if usuario.AuthProviderID == "" {
		t.Error("Execute() should still create the usuario")
	}
	if inviteEmailSent {
		t.Error("Execute() inviteEmailSent = true, want false when the mailer fails")
	}
}

func TestCreateUserNeverSendsInviteWhenPersistenceFails(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{CreateErr: errors.New("insert failed")}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123"}
	mailer := &gatewayfake.Mailer{}
	createUser := NewCreateUser(repository, identity, &gatewayfake.AgencyRepository{}, mailer)

	_, _, err := createUser.Execute(context.Background(), []string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", domain.RolAdministrativo, "")

	if err == nil {
		t.Fatal("Execute() error = nil, want the persistence error")
	}
	if mailer.SendUserInviteCalls != 0 {
		t.Error("Execute() should not send an invite when persistence fails")
	}
}

func TestCreateUserRequiresAgencyForRolInmobiliaria(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{}
	agencies := &gatewayfake.AgencyRepository{}
	createUser := NewCreateUser(repository, identity, agencies, &gatewayfake.Mailer{})

	_, _, err := createUser.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", domain.RolInmobiliaria, "  ")

	if !errors.Is(err, domain.ErrAgenciaRequerida) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgenciaRequerida)
	}
	if identity.CreateCalls != 0 {
		t.Error("Execute() should not call identity provider when the agency is missing")
	}
	if agencies.FindByIDCalls != 0 {
		t.Error("Execute() should not look up an agency when none was sent")
	}
}

func TestCreateUserPropagatesAgencyNotFound(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{}
	agencies := &gatewayfake.AgencyRepository{FindByIDErr: domain.ErrAgencyNotFound}
	createUser := NewCreateUser(repository, identity, agencies, &gatewayfake.Mailer{})

	_, _, err := createUser.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", domain.RolInmobiliaria, "agency-1")

	if !errors.Is(err, domain.ErrAgencyNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgencyNotFound)
	}
	if agencies.FindByIDInput != "agency-1" {
		t.Errorf("Execute() looked up agency %q, want %q", agencies.FindByIDInput, "agency-1")
	}
	if identity.CreateCalls != 0 {
		t.Error("Execute() should not call identity provider when the agency doesn't exist")
	}
}

func TestCreateUserSetsAgencyForRolInmobiliaria(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", InviteURL: "https://app.loteosapp.com/aceptar-invitacion?token_hash=abc"}
	agencies := &gatewayfake.AgencyRepository{FoundByID: domain.Agency{ID: "agency-1"}}
	createUser := NewCreateUser(repository, identity, agencies, &gatewayfake.Mailer{})

	usuario, _, err := createUser.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", domain.RolInmobiliaria, "agency-1")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if usuario.AgencyID == nil || *usuario.AgencyID != "agency-1" {
		t.Errorf("Execute() agency id = %v, want %q", usuario.AgencyID, "agency-1")
	}
	if agencies.FindByIDCalls != 1 {
		t.Errorf("Execute() agencies.FindByID calls = %d, want 1", agencies.FindByIDCalls)
	}
}

func TestCreateUserIgnoresAgencyForOtherRoles(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", InviteURL: "https://app.loteosapp.com/aceptar-invitacion?token_hash=abc"}
	agencies := &gatewayfake.AgencyRepository{}
	createUser := NewCreateUser(repository, identity, agencies, &gatewayfake.Mailer{})

	usuario, _, err := createUser.Execute(context.Background(),
		[]string{domain.RolAdministrador}, "Ana", "Gómez", "ana@example.com", domain.RolAdministrativo, "agency-1")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if usuario.AgencyID != nil {
		t.Errorf("Execute() agency id = %v, want nil for a non-inmobiliaria role", usuario.AgencyID)
	}
	if agencies.FindByIDCalls != 0 {
		t.Error("Execute() should not look up an agency for a role that doesn't have one")
	}
}
