package users

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func administrativoInput() CreateUserInput {
	return CreateUserInput{
		ActorRoles: []string{domain.RolAdministrador},
		Nombre:     "Ana",
		Apellido:   "Gómez",
		Email:      "ana@example.com",
		Rol:        domain.RolAdministrativo,
	}
}

func inmobiliariaInput() CreateUserInput {
	input := administrativoInput()
	input.Rol = domain.RolInmobiliaria
	input.InmobiliariaID = "inm-1"
	return input
}

func TestCreateUserRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	identity := &gatewayfake.IdentityProvider{}
	createUser := NewCreateUser(repository, &gatewayfake.AgencyRepository{}, identity)

	input := administrativoInput()
	input.ActorRoles = []string{"administrativo"}
	_, _, err := createUser.Execute(context.Background(), input)

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
			createUser := NewCreateUser(repository, &gatewayfake.AgencyRepository{}, identity)

			input := administrativoInput()
			input.Nombre = test.nombre
			input.Apellido = test.apellido
			_, _, err := createUser.Execute(context.Background(), input)

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
	createUser := NewCreateUser(repository, &gatewayfake.AgencyRepository{}, identity)

	input := administrativoInput()
	input.Rol = "superadmin"
	_, _, err := createUser.Execute(context.Background(), input)

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
			createUser := NewCreateUser(repository, &gatewayfake.AgencyRepository{}, identity)

			input := administrativoInput()
			input.Rol = rol
			_, _, err := createUser.Execute(context.Background(), input)

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
	createUser := NewCreateUser(repository, &gatewayfake.AgencyRepository{}, identity)

	input := administrativoInput()
	input.Email = "not-an-email"
	_, _, err := createUser.Execute(context.Background(), input)

	if !errors.Is(err, domain.ErrEmailInvalido) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrEmailInvalido)
	}
	if identity.CreateCalls != 0 {
		t.Error("Execute() should not call identity provider when email is invalid")
	}
}

func TestCreateUserRequiresAgencyForInmobiliaria(t *testing.T) {
	t.Parallel()

	for _, inmobiliariaID := range []string{"", "   "} {
		t.Run("inmobiliariaId="+inmobiliariaID, func(t *testing.T) {
			t.Parallel()

			agencies := &gatewayfake.AgencyRepository{}
			identity := &gatewayfake.IdentityProvider{}
			createUser := NewCreateUser(&gatewayfake.UserRepository{}, agencies, identity)

			input := inmobiliariaInput()
			input.InmobiliariaID = inmobiliariaID
			_, _, err := createUser.Execute(context.Background(), input)

			if !errors.Is(err, domain.ErrInmobiliariaRequerida) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInmobiliariaRequerida)
			}
			if agencies.FindByIDCalls != 0 {
				t.Error("Execute() should not look up an agency when none was given")
			}
			if identity.CreateCalls != 0 {
				t.Error("Execute() should not call identity provider without an agency")
			}
		})
	}
}

func TestCreateUserRejectsAgencyForOtherRoles(t *testing.T) {
	t.Parallel()

	for _, rol := range []string{domain.RolAdministrativo, domain.RolEscribano, domain.RolAgrimensor} {
		t.Run(rol, func(t *testing.T) {
			t.Parallel()

			agencies := &gatewayfake.AgencyRepository{}
			identity := &gatewayfake.IdentityProvider{}
			createUser := NewCreateUser(&gatewayfake.UserRepository{}, agencies, identity)

			input := administrativoInput()
			input.Rol = rol
			input.InmobiliariaID = "inm-1"
			_, _, err := createUser.Execute(context.Background(), input)

			if !errors.Is(err, domain.ErrInmobiliariaNoAplica) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrInmobiliariaNoAplica)
			}
			if agencies.FindByIDCalls != 0 {
				t.Error("Execute() should not look up the agency for a role that doesn't belong to one")
			}
			if identity.CreateCalls != 0 {
				t.Error("Execute() should not call identity provider when the agency doesn't apply")
			}
		})
	}
}

func TestCreateUserRejectsUnknownAgency(t *testing.T) {
	t.Parallel()

	agencies := &gatewayfake.AgencyRepository{FindByIDErr: domain.ErrAgencyNotFound}
	identity := &gatewayfake.IdentityProvider{}
	createUser := NewCreateUser(&gatewayfake.UserRepository{}, agencies, identity)

	_, _, err := createUser.Execute(context.Background(), inmobiliariaInput())

	if !errors.Is(err, domain.ErrAgencyNotFound) {
		t.Fatalf("Execute() error = %v, want %v", err, domain.ErrAgencyNotFound)
	}
	if agencies.FindByIDInput != "inm-1" {
		t.Errorf("Execute() looked up agency %q, want %q", agencies.FindByIDInput, "inm-1")
	}
	if identity.CreateCalls != 0 {
		t.Error("Execute() should not call identity provider when the agency doesn't exist")
	}
}

func TestCreateUserWrapsAgencyLookupFailure(t *testing.T) {
	t.Parallel()

	lookupErr := errors.New("connection reset")
	agencies := &gatewayfake.AgencyRepository{FindByIDErr: lookupErr}
	identity := &gatewayfake.IdentityProvider{}
	createUser := NewCreateUser(&gatewayfake.UserRepository{}, agencies, identity)

	_, _, err := createUser.Execute(context.Background(), inmobiliariaInput())

	assertDatabaseUnavailable(t, err, lookupErr)
	if identity.CreateCalls != 0 {
		t.Error("Execute() should not call identity provider when the agency lookup fails")
	}
}

func TestCreateUserLinksInmobiliariaToItsAgency(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	agencies := &gatewayfake.AgencyRepository{FoundByID: domain.Agency{ID: "inm-1", BusinessName: "Lotes del Sur"}}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", TempPassword: "temp-pass-123"}
	createUser := NewCreateUser(repository, agencies, identity)

	input := inmobiliariaInput()
	input.InmobiliariaID = "  inm-1  "
	usuario, _, err := createUser.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if usuario.Rol != domain.RolInmobiliaria {
		t.Errorf("Execute() rol = %q, want %q", usuario.Rol, domain.RolInmobiliaria)
	}
	if usuario.InmobiliariaID == nil || *usuario.InmobiliariaID != "inm-1" {
		t.Errorf("Execute() inmobiliaria id = %v, want %q", usuario.InmobiliariaID, "inm-1")
	}
	if repository.CreateInput.InmobiliariaID == nil || *repository.CreateInput.InmobiliariaID != "inm-1" {
		t.Errorf("Execute() persisted inmobiliaria id = %v, want %q", repository.CreateInput.InmobiliariaID, "inm-1")
	}
}

func TestCreateUserHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	agencies := &gatewayfake.AgencyRepository{}
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", TempPassword: "temp-pass-123"}
	createUser := NewCreateUser(repository, agencies, identity)

	input := administrativoInput()
	input.Nombre = "  Ana  "
	input.Apellido = "  Gómez  "
	input.Email = "  ana@example.com  "
	usuario, tempPassword, err := createUser.Execute(context.Background(), input)
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
	if usuario.InmobiliariaID != nil {
		t.Errorf("Execute() inmobiliaria id = %q, want none for rol %s", *usuario.InmobiliariaID, domain.RolAdministrativo)
	}
	if !usuario.PerfilCompleto {
		t.Error("Execute() should mark the profile as complete when nombre and apellido are given")
	}
	if !usuario.Activo() {
		t.Error("Execute() should create an active user")
	}
	if tempPassword != "temp-pass-123" {
		t.Errorf("Execute() temporary password = %q, want %q", tempPassword, "temp-pass-123")
	}
	if agencies.FindByIDCalls != 0 {
		t.Error("Execute() should not look up an agency for a non-inmobiliaria role")
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
	createUser := NewCreateUser(repository, &gatewayfake.AgencyRepository{}, identity)

	_, _, err := createUser.Execute(context.Background(), administrativoInput())

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
	identity := &gatewayfake.IdentityProvider{AuthProviderID: "sb-123", TempPassword: "temp-pass-123"}
	createUser := NewCreateUser(repository, &gatewayfake.AgencyRepository{}, identity)

	_, _, err := createUser.Execute(context.Background(), administrativoInput())

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
	createUser := NewCreateUser(repository, &gatewayfake.AgencyRepository{}, identity)

	_, _, err := createUser.Execute(context.Background(), administrativoInput())

	assertDatabaseUnavailable(t, err, persistErr)
}
