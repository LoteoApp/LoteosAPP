package gatewayfake

import (
	"context"

	"loteosapp/backend/internal/business/domain"
)

// UserRepository is a fake gateway.UserRepository for tests.
type UserRepository struct {
	CreateCalls int
	CreateErr   error
	Created     domain.Usuario

	UpdateProfileCalls int
	UpdateProfileErr   error
}

func (fake *UserRepository) Create(_ context.Context, usuario domain.Usuario) (domain.Usuario, error) {
	fake.CreateCalls++
	if fake.CreateErr != nil {
		return domain.Usuario{}, fake.CreateErr
	}
	if fake.Created.AuthProviderID == "" {
		return usuario, nil
	}
	return fake.Created, nil
}

func (fake *UserRepository) FindByAuthProviderID(context.Context, string) (domain.Usuario, error) {
	return domain.Usuario{}, nil
}

func (fake *UserRepository) UpdateProfile(_ context.Context, authProviderID, nombre, apellido string) (domain.Usuario, error) {
	fake.UpdateProfileCalls++
	if fake.UpdateProfileErr != nil {
		return domain.Usuario{}, fake.UpdateProfileErr
	}
	return domain.Usuario{AuthProviderID: authProviderID, Nombre: nombre, Apellido: apellido, PerfilCompleto: true}, nil
}
