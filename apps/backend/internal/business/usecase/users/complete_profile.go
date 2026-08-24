package users

import (
	"context"
	"strings"

	"loteosapp/backend/internal/business/domain"
	"loteosapp/backend/internal/business/gateway"
)

// CompleteProfile lets the authenticated user (identified by subject, the
// identity provider's user ID) fill in their own name.
type CompleteProfile interface {
	Execute(ctx context.Context, subject, nombre, apellido string) (domain.Usuario, error)
}

type completeProfileUseCase struct {
	repository gateway.UserRepository
}

func NewCompleteProfile(repository gateway.UserRepository) CompleteProfile {
	return &completeProfileUseCase{repository: repository}
}

func (useCase *completeProfileUseCase) Execute(ctx context.Context, subject, nombre, apellido string) (domain.Usuario, error) {
	nombre = strings.TrimSpace(nombre)
	apellido = strings.TrimSpace(apellido)
	if nombre == "" || apellido == "" {
		return domain.Usuario{}, domain.ErrPerfilInvalido
	}

	return useCase.repository.UpdateProfile(ctx, subject, nombre, apellido)
}
