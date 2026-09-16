package users

import (
	"context"
	"errors"
	"testing"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway/gatewayfake"
)

func TestCompleteProfileHappyPath(t *testing.T) {
	t.Parallel()

	repository := &gatewayfake.UserRepository{}
	completeProfile := NewCompleteProfile(repository)

	usuario, err := completeProfile.Execute(context.Background(), "sb-123", "Ana", "Gómez")
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !usuario.PerfilCompleto {
		t.Error("Execute() should mark the profile as complete")
	}
	if usuario.Nombre != "Ana" || usuario.Apellido != "Gómez" {
		t.Errorf("Execute() usuario = %#v", usuario)
	}
	if repository.UpdateProfileCalls != 1 {
		t.Errorf("Execute() repository.UpdateProfile calls = %d, want 1", repository.UpdateProfileCalls)
	}
}

func TestCompleteProfileRejectsEmptyFields(t *testing.T) {
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

			repository := &gatewayfake.UserRepository{}
			completeProfile := NewCompleteProfile(repository)

			_, err := completeProfile.Execute(context.Background(), "sb-123", test.nombre, test.apellido)

			if !errors.Is(err, domain.ErrPerfilInvalido) {
				t.Fatalf("Execute() error = %v, want %v", err, domain.ErrPerfilInvalido)
			}
			if repository.UpdateProfileCalls != 0 {
				t.Error("Execute() should not call repository with invalid input")
			}
		})
	}
}

func TestCompleteProfilePropagatesRepositoryError(t *testing.T) {
	t.Parallel()

	wantErr := domain.ErrUsuarioNoEncontrado
	repository := &gatewayfake.UserRepository{UpdateProfileErr: wantErr}
	completeProfile := NewCompleteProfile(repository)

	_, err := completeProfile.Execute(context.Background(), "sb-123", "Ana", "Gómez")

	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}
