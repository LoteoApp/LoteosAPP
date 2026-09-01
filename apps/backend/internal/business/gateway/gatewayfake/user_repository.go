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
	CreateInput domain.Usuario

	FindByAuthProviderIDCalls   int
	FindByAuthProviderIDErr     error
	FindByAuthProviderIDResult  domain.Usuario
	FindByAuthProviderIDSubject string

	FindByIDCalls int
	FindByIDErr   error
	FoundByID     domain.Usuario
	FindByIDInput string

	UpdateProfileCalls int
	UpdateProfileErr   error

	ListByRolCalls    int
	ListByRolErr      error
	ListByRolResult   []domain.Usuario
	ListByRolInput    domain.Rol
	ListByRolInactive bool

	UpdateCalls int
	UpdateErr   error
	Updated     domain.Usuario
	UpdateInput domain.UsuarioUpdate

	SoftDeleteCalls  int
	SoftDeleteErr    error
	SoftDeletedID    string
	SoftDeletedActor string
}

func (fake *UserRepository) Create(_ context.Context, usuario domain.Usuario) (domain.Usuario, error) {
	fake.CreateCalls++
	fake.CreateInput = usuario
	if fake.CreateErr != nil {
		return domain.Usuario{}, fake.CreateErr
	}
	if fake.Created.AuthProviderID == "" {
		return usuario, nil
	}
	return fake.Created, nil
}

func (fake *UserRepository) FindByAuthProviderID(_ context.Context, authProviderID string) (domain.Usuario, error) {
	fake.FindByAuthProviderIDCalls++
	fake.FindByAuthProviderIDSubject = authProviderID
	if fake.FindByAuthProviderIDErr != nil {
		return domain.Usuario{}, fake.FindByAuthProviderIDErr
	}
	return fake.FindByAuthProviderIDResult, nil
}

func (fake *UserRepository) FindByID(_ context.Context, id string) (domain.Usuario, error) {
	fake.FindByIDCalls++
	fake.FindByIDInput = id
	if fake.FindByIDErr != nil {
		return domain.Usuario{}, fake.FindByIDErr
	}
	return fake.FoundByID, nil
}

func (fake *UserRepository) UpdateProfile(_ context.Context, authProviderID, nombre, apellido string) (domain.Usuario, error) {
	fake.UpdateProfileCalls++
	if fake.UpdateProfileErr != nil {
		return domain.Usuario{}, fake.UpdateProfileErr
	}
	return domain.Usuario{AuthProviderID: authProviderID, Nombre: nombre, Apellido: apellido, PerfilCompleto: true}, nil
}

func (fake *UserRepository) ListByRol(_ context.Context, rol domain.Rol, includeInactive bool) ([]domain.Usuario, error) {
	fake.ListByRolCalls++
	fake.ListByRolInput = rol
	fake.ListByRolInactive = includeInactive
	if fake.ListByRolErr != nil {
		return nil, fake.ListByRolErr
	}
	return fake.ListByRolResult, nil
}

func (fake *UserRepository) Update(_ context.Context, update domain.UsuarioUpdate) (domain.Usuario, error) {
	fake.UpdateCalls++
	fake.UpdateInput = update
	if fake.UpdateErr != nil {
		return domain.Usuario{}, fake.UpdateErr
	}
	if fake.Updated.ID == "" {
		usuario := domain.Usuario{ID: update.ID, Rol: domain.RolAgrimensor}
		if update.Nombre != nil {
			usuario.Nombre = *update.Nombre
		}
		if update.Apellido != nil {
			usuario.Apellido = *update.Apellido
		}
		return usuario, nil
	}
	return fake.Updated, nil
}

func (fake *UserRepository) SoftDelete(_ context.Context, id, usuarioModificacion string) error {
	fake.SoftDeleteCalls++
	fake.SoftDeletedID = id
	fake.SoftDeletedActor = usuarioModificacion
	return fake.SoftDeleteErr
}
