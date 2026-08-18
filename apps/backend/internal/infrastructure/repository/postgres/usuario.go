package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"loteosapp/backend/internal/business/domain"
)

const uniqueViolationCode = "23505"

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (repository *UserRepository) Create(ctx context.Context, usuario domain.Usuario) (domain.Usuario, error) {
	var created domain.Usuario

	err := repository.pool.QueryRow(ctx, `
		INSERT INTO usuarios (auth_provider_id, email, rol)
		VALUES ($1::uuid, $2, $3)
		RETURNING id::text, auth_provider_id::text, email, nombre, apellido, rol, perfil_completo, created_at
	`, usuario.AuthProviderID, usuario.Email, usuario.Rol).Scan(
		&created.ID, &created.AuthProviderID, &created.Email, &created.Nombre,
		&created.Apellido, &created.Rol, &created.PerfilCompleto, &created.CreatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return domain.Usuario{}, domain.ErrEmailEnUso
		}
		return domain.Usuario{}, err
	}

	return created, nil
}

func (repository *UserRepository) FindByAuthProviderID(ctx context.Context, authProviderID string) (domain.Usuario, error) {
	var usuario domain.Usuario

	err := repository.pool.QueryRow(ctx, `
		SELECT id::text, auth_provider_id::text, email, nombre, apellido, rol, perfil_completo, created_at
		FROM usuarios
		WHERE auth_provider_id = $1::uuid
	`, authProviderID).Scan(
		&usuario.ID, &usuario.AuthProviderID, &usuario.Email, &usuario.Nombre,
		&usuario.Apellido, &usuario.Rol, &usuario.PerfilCompleto, &usuario.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
	}
	if err != nil {
		return domain.Usuario{}, err
	}

	return usuario, nil
}

func (repository *UserRepository) UpdateProfile(ctx context.Context, authProviderID, nombre, apellido string) (domain.Usuario, error) {
	var usuario domain.Usuario

	err := repository.pool.QueryRow(ctx, `
		UPDATE usuarios
		SET nombre = $2, apellido = $3, perfil_completo = true, updated_at = now()
		WHERE auth_provider_id = $1::uuid
		RETURNING id::text, auth_provider_id::text, email, nombre, apellido, rol, perfil_completo, created_at
	`, authProviderID, nombre, apellido).Scan(
		&usuario.ID, &usuario.AuthProviderID, &usuario.Email, &usuario.Nombre,
		&usuario.Apellido, &usuario.Rol, &usuario.PerfilCompleto, &usuario.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Usuario{}, domain.ErrUsuarioNoEncontrado
	}
	if err != nil {
		return domain.Usuario{}, err
	}

	return usuario, nil
}
