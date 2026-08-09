package usecase

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
)

type userRepositoryFake struct {
	createCalls int
	createErr   error
	created     domain.Usuario

	updateProfileCalls int
	updateProfileErr   error
	updatedProfile     domain.Usuario
}

func (fake *userRepositoryFake) Create(_ context.Context, usuario domain.Usuario) (domain.Usuario, error) {
	fake.createCalls++
	if fake.createErr != nil {
		return domain.Usuario{}, fake.createErr
	}
	if fake.created.KeycloakID == "" {
		return usuario, nil
	}
	return fake.created, nil
}

func (fake *userRepositoryFake) FindByKeycloakID(context.Context, string) (domain.Usuario, error) {
	return domain.Usuario{}, nil
}

func (fake *userRepositoryFake) UpdateProfile(_ context.Context, keycloakID, nombre, apellido string) (domain.Usuario, error) {
	fake.updateProfileCalls++
	if fake.updateProfileErr != nil {
		return domain.Usuario{}, fake.updateProfileErr
	}
	return domain.Usuario{KeycloakID: keycloakID, Nombre: nombre, Apellido: apellido, PerfilCompleto: true}, nil
}

type identityProviderFake struct {
	createCalls int
	createErr   error
	keycloakID  string
	tempPass    string

	deleteCalls   int
	deleteErr     error
	deletedUserID string
}

func (fake *identityProviderFake) CreateUser(_ context.Context, email, rol string) (string, string, error) {
	fake.createCalls++
	if fake.createErr != nil {
		return "", "", fake.createErr
	}
	return fake.keycloakID, fake.tempPass, nil
}

func (fake *identityProviderFake) DeleteUser(_ context.Context, keycloakID string) error {
	fake.deleteCalls++
	fake.deletedUserID = keycloakID
	return fake.deleteErr
}

func TestUserServiceCreateUserRejectsNonAdministrador(t *testing.T) {
	t.Parallel()

	repository := &userRepositoryFake{}
	identity := &identityProviderFake{}
	service := NewUserService(repository, identity)

	_, _, err := service.CreateUser(context.Background(), []string{"administrativo"}, "ana@example.com", domain.RolAdministrativo)

	if !errors.Is(err, domain.ErrNoAutorizado) {
		t.Fatalf("CreateUser() error = %v, want %v", err, domain.ErrNoAutorizado)
	}
	if identity.createCalls != 0 {
		t.Error("CreateUser() should not call identity provider when actor is not administrador")
	}
	if repository.createCalls != 0 {
		t.Error("CreateUser() should not call repository when actor is not administrador")
	}
}

func TestUserServiceCreateUserRejectsInvalidRol(t *testing.T) {
	t.Parallel()

	repository := &userRepositoryFake{}
	identity := &identityProviderFake{}
	service := NewUserService(repository, identity)

	_, _, err := service.CreateUser(context.Background(), []string{domain.RolAdministrador}, "ana@example.com", "superadmin")

	if !errors.Is(err, domain.ErrRolInvalido) {
		t.Fatalf("CreateUser() error = %v, want %v", err, domain.ErrRolInvalido)
	}
	if identity.createCalls != 0 {
		t.Error("CreateUser() should not call identity provider when rol is invalid")
	}
}

func TestUserServiceCreateUserRejectsInvalidEmail(t *testing.T) {
	t.Parallel()

	repository := &userRepositoryFake{}
	identity := &identityProviderFake{}
	service := NewUserService(repository, identity)

	_, _, err := service.CreateUser(context.Background(), []string{domain.RolAdministrador}, "not-an-email", domain.RolAdministrativo)

	if !errors.Is(err, domain.ErrEmailInvalido) {
		t.Fatalf("CreateUser() error = %v, want %v", err, domain.ErrEmailInvalido)
	}
	if identity.createCalls != 0 {
		t.Error("CreateUser() should not call identity provider when email is invalid")
	}
}

func TestUserServiceCreateUserHappyPath(t *testing.T) {
	t.Parallel()

	repository := &userRepositoryFake{}
	identity := &identityProviderFake{keycloakID: "kc-123", tempPass: "temp-pass-123"}
	service := NewUserService(repository, identity)

	usuario, tempPassword, err := service.CreateUser(context.Background(), []string{domain.RolAdministrador}, "ana@example.com", domain.RolAdministrativo)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if usuario.KeycloakID != "kc-123" {
		t.Errorf("CreateUser() keycloak id = %q, want %q", usuario.KeycloakID, "kc-123")
	}
	if usuario.Email != "ana@example.com" {
		t.Errorf("CreateUser() email = %q, want %q", usuario.Email, "ana@example.com")
	}
	if tempPassword != "temp-pass-123" {
		t.Errorf("CreateUser() temporary password = %q, want %q", tempPassword, "temp-pass-123")
	}
	if repository.createCalls != 1 {
		t.Errorf("CreateUser() repository.Create calls = %d, want 1", repository.createCalls)
	}
	if identity.deleteCalls != 0 {
		t.Error("CreateUser() should not compensate when persistence succeeds")
	}
}

func TestUserServiceCreateUserPropagatesIdentityProviderError(t *testing.T) {
	t.Parallel()

	repository := &userRepositoryFake{}
	identity := &identityProviderFake{createErr: domain.ErrEmailEnUso}
	service := NewUserService(repository, identity)

	_, _, err := service.CreateUser(context.Background(), []string{domain.RolAdministrador}, "ana@example.com", domain.RolAdministrativo)

	if !errors.Is(err, domain.ErrEmailEnUso) {
		t.Fatalf("CreateUser() error = %v, want %v", err, domain.ErrEmailEnUso)
	}
	if repository.createCalls != 0 {
		t.Error("CreateUser() should not call repository when identity provider fails")
	}
}

func TestUserServiceCreateUserCompensatesWhenPersistenceFails(t *testing.T) {
	t.Parallel()

	persistErr := errors.New("insert failed")
	repository := &userRepositoryFake{createErr: persistErr}
	identity := &identityProviderFake{keycloakID: "kc-123", tempPass: "temp-pass-123"}
	service := NewUserService(repository, identity)

	_, _, err := service.CreateUser(context.Background(), []string{domain.RolAdministrador}, "ana@example.com", domain.RolAdministrativo)

	if !errors.Is(err, persistErr) {
		t.Fatalf("CreateUser() error = %v, want %v", err, persistErr)
	}
	if identity.deleteCalls != 1 {
		t.Fatalf("CreateUser() identity.DeleteUser calls = %d, want 1", identity.deleteCalls)
	}
	if identity.deletedUserID != "kc-123" {
		t.Errorf("CreateUser() compensated user id = %q, want %q", identity.deletedUserID, "kc-123")
	}
}

func TestUserServiceCreateUserReturnsOriginalErrorWhenCompensationAlsoFails(t *testing.T) {
	t.Parallel()

	persistErr := errors.New("insert failed")
	repository := &userRepositoryFake{createErr: persistErr}
	identity := &identityProviderFake{keycloakID: "kc-123", deleteErr: errors.New("delete failed")}
	service := NewUserService(repository, identity)

	_, _, err := service.CreateUser(context.Background(), []string{domain.RolAdministrador}, "ana@example.com", domain.RolAdministrativo)

	if !errors.Is(err, persistErr) {
		t.Fatalf("CreateUser() error = %v, want %v", err, persistErr)
	}
}

func TestUserServiceCompleteProfileHappyPath(t *testing.T) {
	t.Parallel()

	repository := &userRepositoryFake{}
	service := NewUserService(repository, &identityProviderFake{})

	usuario, err := service.CompleteProfile(context.Background(), "kc-123", "Ana", "Gómez")
	if err != nil {
		t.Fatalf("CompleteProfile() error = %v", err)
	}
	if !usuario.PerfilCompleto {
		t.Error("CompleteProfile() should mark the profile as complete")
	}
	if usuario.Nombre != "Ana" || usuario.Apellido != "Gómez" {
		t.Errorf("CompleteProfile() usuario = %#v", usuario)
	}
	if repository.updateProfileCalls != 1 {
		t.Errorf("CompleteProfile() repository.UpdateProfile calls = %d, want 1", repository.updateProfileCalls)
	}
}

func TestUserServiceCompleteProfileRejectsEmptyFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		nombre   string
		apellido string
	}{
		{name: "empty nombre", nombre: "  ", apellido: "Gómez"},
		{name: "empty apellido", nombre: "Ana", apellido: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			repository := &userRepositoryFake{}
			service := NewUserService(repository, &identityProviderFake{})

			_, err := service.CompleteProfile(context.Background(), "kc-123", test.nombre, test.apellido)

			if !errors.Is(err, domain.ErrPerfilInvalido) {
				t.Fatalf("CompleteProfile() error = %v, want %v", err, domain.ErrPerfilInvalido)
			}
			if repository.updateProfileCalls != 0 {
				t.Error("CompleteProfile() should not call repository with invalid input")
			}
		})
	}
}

func TestUserServiceCompleteProfilePropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrUsuarioNoEncontrado
	repository := &userRepositoryFake{updateProfileErr: wantErr}
	service := NewUserService(repository, &identityProviderFake{})

	_, err := service.CompleteProfile(context.Background(), "kc-123", "Ana", "Gómez")

	if !errors.Is(err, wantErr) {
		t.Fatalf("CompleteProfile() error = %v, want %v", err, wantErr)
	}
}
